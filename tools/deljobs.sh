#!/bin/sh

for j in $(kubectl get jobs -o custom-columns=:.metadata.name)
do
	kubectl delete jobs $j &
done
