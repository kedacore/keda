package kusto

import (
	"fmt"
	"github.com/Azure/azure-kusto-go/kusto/internal/version"
	"github.com/Azure/azure-kusto-go/kusto/utils"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/samber/lo"
)

type ClientDetails struct {
	// applicationForTracing is the name of the application that is using the client.
	applicationForTracing string
	// userNameForTracing is the name of the user that is using the client.
	userNameForTracing string
	// clientVersionForTracing is the version of the client.
	clientVersionForTracing string
}

func NewClientDetails(applicationForTracing string, userNameForTracing string) *ClientDetails {
	return &ClientDetails{applicationForTracing: applicationForTracing, userNameForTracing: userNameForTracing}
}

type StringPair struct {
	Key   string
	Value string
}

const NONE = "[none]"

var defaultTracingValuesOnce = utils.NewOnceWithInit[ClientDetails](func() (ClientDetails, error) {
	return ClientDetails{
		applicationForTracing:   filepath.Base(os.Args[0]),
		userNameForTracing:      getOsUser(),
		clientVersionForTracing: buildHeaderFormat(StringPair{Key: "Kusto.Go.Client", Value: version.Kusto}, StringPair{Key: "Runtime.Go", Value: runtime.Version()}),
	}, nil
})

func getOsUser() string {
	var final string
	current, err := user.Current()
	if err == nil && current.Username != "" {
		final = current.Username
	} else {
		// get from env and try domain too
		final = os.Getenv("USERNAME")
		domain := os.Getenv("USERDOMAIN")
		if !isEmpty(domain) && !isEmpty(final) {
			final = domain + "\\" + final
		}
	}

	if isEmpty(final) {
		final = NONE
	}

	return final
}

var escapeRegex = regexp.MustCompile("[\\r\\n\\s{}|]+")

func escape(s string) string {
	return "{" + escapeRegex.ReplaceAllString(s, "_") + "}"
}

func defaultTracingValues() ClientDetails {
	r, _ := defaultTracingValuesOnce.DoWithInit()
	return r
}

func (c *ClientDetails) ApplicationForTracing() string {
	if c.applicationForTracing == "" {
		return defaultTracingValues().applicationForTracing
	}
	return c.applicationForTracing
}

func (c *ClientDetails) UserNameForTracing() string {
	if c.userNameForTracing == "" {
		return defaultTracingValues().userNameForTracing
	}
	return c.userNameForTracing
}

func (c *ClientDetails) ClientVersionForTracing() string {
	return defaultTracingValues().clientVersionForTracing
}

func buildHeaderFormat(args ...StringPair) string {
	return strings.Join(lo.Map(args, func(arg StringPair, _ int) string {
		return fmt.Sprintf("%s:%s", arg.Key, escape(arg.Value))
	}), "|")
}

func setConnectorDetails(name, version, appName, appVersion string, sendUser bool, overrideUser string, additionalFields ...StringPair) (string, string) {
	var additionalFieldsList []StringPair

	additionalFieldsList = append(additionalFieldsList, StringPair{Key: "Kusto." + name, Value: version})

	if appName == "" {
		appName = defaultTracingValues().applicationForTracing
	}
	if appVersion == "" {
		appVersion = NONE
	}

	additionalFieldsList = append(additionalFieldsList, StringPair{Key: "App.{" + appName + "}", Value: appVersion})
	if additionalFields != nil {
		additionalFieldsList = append(additionalFieldsList, additionalFields...)
	}

	app := buildHeaderFormat(additionalFieldsList...)

	user := NONE

	if sendUser {
		if overrideUser != "" {
			user = overrideUser
		} else {
			user = defaultTracingValues().userNameForTracing
		}
	}

	return app, user
}
