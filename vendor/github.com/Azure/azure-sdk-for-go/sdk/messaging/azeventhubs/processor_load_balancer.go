// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package azeventhubs

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/internal/log"
)

type processorLoadBalancer struct {
	checkpointStore             CheckpointStore
	details                     consumerClientDetails
	strategy                    ProcessorStrategy
	partitionExpirationDuration time.Duration

	// NOTE: when you create your own *rand.Rand it is not thread safe.
	rnd *rand.Rand
}

func newProcessorLoadBalancer(checkpointStore CheckpointStore, details consumerClientDetails, strategy ProcessorStrategy, partitionExpiration time.Duration) *processorLoadBalancer {
	return &processorLoadBalancer{
		checkpointStore:             checkpointStore,
		details:                     details,
		strategy:                    strategy,
		partitionExpirationDuration: partitionExpiration,
		rnd:                         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

type loadBalancerInfo struct {
	// current are the partitions that _we_ own
	current []Ownership

	// unownedOrExpired partitions either had no claim _ever_ or were once
	// owned but the ownership claim has expired.
	unownedOrExpired []Ownership

	// aboveMax are ownerships where the specific owner has too many partitions
	// it contains _all_ the partitions for that particular consumer.
	aboveMax []Ownership

	// claimMorePartitions is true when we should try to claim more partitions
	// because we're under the limit, or we're in a situation where we could claim
	// one extra partition.
	claimMorePartitions bool

	// maxAllowed is the maximum number of partitions that other processors are allowed
	// to own during this round. It can change based on how many partitions we own and whether
	// an 'extra' partition is allowed (ie, partitions %owners is not 0). Look at
	// [processorLoadBalancer.getAvailablePartitions] for more details.
	maxAllowed int

	raw []Ownership
}

// loadBalance calls through to the user's configured load balancing algorithm.
// NOTE: this function is NOT thread safe!
func (lb *processorLoadBalancer) LoadBalance(ctx context.Context, partitionIDs []string) ([]Ownership, error) {
	lbinfo, err := lb.getAvailablePartitions(ctx, partitionIDs)

	if err != nil {
		return nil, err
	}

	// if we don't need to claim any more partitions we'll just keep reclaiming the partitions we currently have
	ownerships := lbinfo.current

	if lbinfo.claimMorePartitions {
		switch lb.strategy {
		case ProcessorStrategyGreedy:
			log.Writef(EventConsumer, "[%s] Using greedy strategy to claim partitions", lb.details.ClientID)
			ownerships = lb.greedyLoadBalancer(ctx, lbinfo)
		case ProcessorStrategyBalanced:
			log.Writef(EventConsumer, "[%s] Using balanced strategy to claim partitions", lb.details.ClientID)

			o := lb.balancedLoadBalancer(ctx, lbinfo)

			if o != nil {
				ownerships = append(ownerships, *o)
			}
		default:
			return nil, fmt.Errorf("[%s] invalid load balancing strategy '%s'", lb.details.ClientID, lb.strategy)
		}
	}

	actual, err := lb.checkpointStore.ClaimOwnership(ctx, ownerships, nil)

	if err != nil {
		return nil, err
	}

	if log.Should(EventConsumer) {
		log.Writef(EventConsumer, "[%0.5s] Asked for %s, got %s", lb.details.ClientID, partitionsForOwnerships(ownerships), partitionsForOwnerships(actual))
	}

	return actual, nil
}

func partitionsForOwnerships(all []Ownership) string {
	var parts []string

	for _, o := range all {
		parts = append(parts, o.PartitionID)
	}

	return strings.Join(parts, ",")
}

// getAvailablePartitions looks through the ownership list (using the checkpointstore.ListOwnership) and evaluates:
//   - Whether we should claim more partitions
//   - Which partitions are available - unowned/relinquished, expired or processors that own more than the maximum allowed.
//
// Load balancing happens in individual functions
func (lb *processorLoadBalancer) getAvailablePartitions(ctx context.Context, partitionIDs []string) (loadBalancerInfo, error) {
	log.Writef(EventConsumer, "[%s] Listing ownership for %s/%s/%s", lb.details.ClientID, lb.details.FullyQualifiedNamespace, lb.details.EventHubName, lb.details.ConsumerGroup)

	ownerships, err := lb.checkpointStore.ListOwnership(ctx, lb.details.FullyQualifiedNamespace, lb.details.EventHubName, lb.details.ConsumerGroup, nil)

	if err != nil {
		return loadBalancerInfo{}, err
	}

	alreadyAdded := map[string]bool{}
	groupedByOwner := map[string][]Ownership{
		lb.details.ClientID: nil,
	}

	var unownedOrExpired []Ownership

	// split out partitions by whether they're currently owned
	// and if they're expired/relinquished.
	for _, o := range ownerships {
		alreadyAdded[o.PartitionID] = true

		if time.Since(o.LastModifiedTime.UTC()) > lb.partitionExpirationDuration {
			unownedOrExpired = append(unownedOrExpired, o)
			continue
		}

		if o.OwnerID == relinquishedOwnershipID {
			unownedOrExpired = append(unownedOrExpired, o)
			continue
		}

		groupedByOwner[o.OwnerID] = append(groupedByOwner[o.OwnerID], o)
	}

	numExpired := len(unownedOrExpired)

	// add in all the unowned partitions
	for _, partID := range partitionIDs {
		if alreadyAdded[partID] {
			continue
		}

		unownedOrExpired = append(unownedOrExpired, Ownership{
			FullyQualifiedNamespace: lb.details.FullyQualifiedNamespace,
			ConsumerGroup:           lb.details.ConsumerGroup,
			EventHubName:            lb.details.EventHubName,
			PartitionID:             partID,
			OwnerID:                 lb.details.ClientID,
			// note that we don't have etag info here since nobody has
			// ever owned this partition.
		})
	}

	minRequired := len(partitionIDs) / len(groupedByOwner)
	maxAllowed := minRequired
	allowExtraPartition := len(partitionIDs)%len(groupedByOwner) > 0

	// only allow owners to keep extra partitions if we've already met our minimum bar. Otherwise
	// above the minimum is fair game.
	if allowExtraPartition && len(groupedByOwner[lb.details.ClientID]) >= minRequired {
		maxAllowed += 1
	}

	var aboveMax []Ownership

	for id, ownerships := range groupedByOwner {
		if id == lb.details.ClientID {
			continue
		}

		if len(ownerships) > maxAllowed {
			aboveMax = append(aboveMax, ownerships...)
		}
	}

	claimMorePartitions := true
	current := groupedByOwner[lb.details.ClientID]

	if len(current) >= maxAllowed {
		// - I have _exactly_ the right amount
		// or
		// - I have too many. We expect to have some stolen from us, but we'll maintain
		//    ownership for now.
		claimMorePartitions = false
	} else if allowExtraPartition && len(current) == maxAllowed-1 {
		// In the 'allowExtraPartition' scenario, some consumers will have an extra partition
		// since things don't divide up evenly. We're one under the max, which means we _might_
		// be able to claim another one.
		//
		// We will attempt to grab _one_ more but only if there are free partitions available
		// or if one of the consumers has more than the max allowed.
		claimMorePartitions = len(unownedOrExpired) > 0 || len(aboveMax) > 0
	}

	log.Writef(EventConsumer, "[%s] claimMorePartitions: %t, owners: %d, current: %d, unowned: %d, expired: %d, above: %d",
		lb.details.ClientID,
		claimMorePartitions,
		len(groupedByOwner),
		len(current),
		len(unownedOrExpired)-numExpired,
		numExpired,
		len(aboveMax))

	return loadBalancerInfo{
		current:             current,
		unownedOrExpired:    unownedOrExpired,
		aboveMax:            aboveMax,
		claimMorePartitions: claimMorePartitions,
		raw:                 ownerships,
		maxAllowed:          maxAllowed,
	}, nil
}

// greedyLoadBalancer will attempt to grab as many free partitions as it needs to balance
// in each round.
func (lb *processorLoadBalancer) greedyLoadBalancer(ctx context.Context, lbinfo loadBalancerInfo) []Ownership {
	ours := lbinfo.current

	// try claiming from the completely unowned or expires ownerships _first_
	randomOwnerships := getRandomOwnerships(lb.rnd, lbinfo.unownedOrExpired, lbinfo.maxAllowed-len(ours))
	ours = append(ours, randomOwnerships...)

	if len(ours) < lbinfo.maxAllowed {
		log.Writef(EventConsumer, "Not enough expired or unowned partitions, will need to steal from other processors")

		// if that's not enough then we'll randomly steal from any owners that had partitions
		// above the maximum.
		randomOwnerships := getRandomOwnerships(lb.rnd, lbinfo.aboveMax, lbinfo.maxAllowed-len(ours))
		ours = append(ours, randomOwnerships...)
	}

	for i := 0; i < len(ours); i++ {
		ours[i] = lb.resetOwnership(ours[i])
	}

	return ours
}

// balancedLoadBalancer attempts to split the partition load out between the available
// consumers so each one has an even amount (or even + 1, if the # of consumers and #
// of partitions doesn't divide evenly).
//
// NOTE: the checkpoint store itself does not have a concept of 'presence' that doesn't
// ALSO involve owning a partition. It's possible for a consumer to get boxed out for a
// bit until it manages to steal at least one partition since the other consumers don't
// know it exists until then.
func (lb *processorLoadBalancer) balancedLoadBalancer(ctx context.Context, lbinfo loadBalancerInfo) *Ownership {
	if len(lbinfo.unownedOrExpired) > 0 {
		idx := lb.rnd.Intn(len(lbinfo.unownedOrExpired))
		o := lb.resetOwnership(lbinfo.unownedOrExpired[idx])
		return &o
	}

	if len(lbinfo.aboveMax) > 0 {
		idx := lb.rnd.Intn(len(lbinfo.aboveMax))
		o := lb.resetOwnership(lbinfo.aboveMax[idx])
		return &o
	}

	return nil
}

func (lb *processorLoadBalancer) resetOwnership(o Ownership) Ownership {
	o.OwnerID = lb.details.ClientID
	return o
}

func getRandomOwnerships(rnd *rand.Rand, ownerships []Ownership, count int) []Ownership {
	limit := int(math.Min(float64(count), float64(len(ownerships))))

	if limit == 0 {
		return nil
	}

	choices := rnd.Perm(limit)

	var newOwnerships []Ownership

	for i := 0; i < len(choices); i++ {
		newOwnerships = append(newOwnerships, ownerships[choices[i]])
	}

	return newOwnerships
}
