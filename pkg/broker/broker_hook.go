package broker

import (
	"bytes"
	"regexp"
	"strings"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"

	"github.com/iotaledger/hive.go/ierrors"
	"github.com/iotaledger/hive.go/web/basicauth"
	"github.com/iotaledger/hive.go/web/subscriptionmanager"
)

// BrokerHook is the glue code between the mochi-mqtt server and inx-mqtt.
// It is responsible for keeping the subscription manager up to date.
// It also handles authorization and protection of topics.
//
//nolint:revive
type BrokerHook struct {
	mqtt.HookBase

	subscriptionManager  *subscriptionmanager.SubscriptionManager[string, string]
	basicAuthManager     *basicauth.BasicAuthManager
	matchPublicTopics    func(topic string) bool
	matchProtectedTopics func(topic string) bool
}

func NewBrokerHook(subscriptionManager *subscriptionmanager.SubscriptionManager[string, string],
	basicAuthManager *basicauth.BasicAuthManager,
	publicTopics []string,
	protectedTopics []string) (*BrokerHook, error) {

	regexesPublicTopics, err := compileTopicsAsRegexes(publicTopics)
	if err != nil {
		return nil, ierrors.Wrap(err, "failed to compile public topics")
	}

	regexesProtectedTopics, err := compileTopicsAsRegexes(protectedTopics)
	if err != nil {
		return nil, ierrors.Wrap(err, "failed to compile protected topics")
	}

	matchPublicTopics := func(topic string) bool {
		loweredTopic := strings.ToLower(topic)

		for _, reg := range regexesPublicTopics {
			if reg.MatchString(loweredTopic) {
				return true
			}
		}

		return false
	}

	matchProtectedTopics := func(topic string) bool {
		loweredTopic := strings.ToLower(topic)

		for _, reg := range regexesProtectedTopics {
			if reg.MatchString(loweredTopic) {
				return true
			}
		}

		return false
	}

	return &BrokerHook{
		subscriptionManager:  subscriptionManager,
		basicAuthManager:     basicAuthManager,
		matchPublicTopics:    matchPublicTopics,
		matchProtectedTopics: matchProtectedTopics,
	}, nil
}

// ID returns the ID of the hook.
func (h *BrokerHook) ID() string {
	return "iota-mqtt-broker-hook"
}

// Provides indicates which methods a hook provides. The default is none - this method
// should be overridden by the embedding hook.
func (h *BrokerHook) Provides(b byte) bool {
	//nolint:gocritic // false positive, the argument order is correct
	return bytes.Contains([]byte{
		mqtt.OnConnectAuthenticate,
		mqtt.OnACLCheck,
		mqtt.OnSessionEstablished,
		mqtt.OnDisconnect,
		mqtt.OnSubscribed,
		mqtt.OnUnsubscribed,
	}, []byte{b})
}

// OnConnectAuthenticate returns true if the connecting client has rules which provide access
// in the auth ledger.
func (h *BrokerHook) OnConnectAuthenticate(cl *mqtt.Client, pk packets.Packet) bool {
	username := string(cl.Properties.Username)
	password := pk.Connect.Password

	// check if the given user exists in the users map
	if !h.basicAuthManager.Exists(username) {
		// if the user doesn't exist, the user is still allowed to connect, but without access to the protected topics.
		return true
	}

	// if the user exists in the users map, we need to check if the given password is correct, otherwise the user is not allowed to connect.
	// All successfully connected users which are know in the users map automatically get elevated priviledges to access protected topics.
	return h.basicAuthManager.VerifyUsernameAndPasswordBytes(username, password)
}

// OnACLCheck returns true if the connecting client has matching read or write access to subscribe
// or publish to a given topic.
func (h *BrokerHook) OnACLCheck(cl *mqtt.Client, topic string, write bool) bool {
	// clients are not allowed to write
	if write {
		return false
	}

	// check if the topic is protected
	if h.matchProtectedTopics(topic) {
		// check if the client is authenticated
		// if the client is not authenticated, it is not allowed to access protected topics
		return h.basicAuthManager.Exists(string(cl.Properties.Username))
	}

	// allow everyone to read public topics
	return h.matchPublicTopics(topic)
}

// OnSessionEstablished is called when a new client establishes a session (after OnConnect).
func (h *BrokerHook) OnSessionEstablished(cl *mqtt.Client, _ packets.Packet) {
	h.subscriptionManager.Connect(cl.ID)
}

// OnDisconnect is called when a client is disconnected for any reason.
func (h *BrokerHook) OnDisconnect(cl *mqtt.Client, _ error, _ bool) {
	h.subscriptionManager.Disconnect(cl.ID)
}

// OnSubscribed is called when a client subscribes to one or more filters.
func (h *BrokerHook) OnSubscribed(cl *mqtt.Client, pk packets.Packet, _ []byte) {
	for _, subscription := range pk.Filters {
		h.subscriptionManager.Subscribe(cl.ID, subscription.Filter)
	}
}

// OnUnsubscribed is called when a client unsubscribes from one or more filters.
func (h *BrokerHook) OnUnsubscribed(cl *mqtt.Client, pk packets.Packet) {
	for _, subscription := range pk.Filters {
		h.subscriptionManager.Unsubscribe(cl.ID, subscription.Filter)
	}
}

func compileTopicAsRegex(topic string) *regexp.Regexp {

	r := regexp.QuoteMeta(topic)
	r = strings.ReplaceAll(r, `\*`, "(.*?)")
	r += "$"

	reg, err := regexp.Compile(r)
	if err != nil {
		return nil
	}

	return reg
}

func compileTopicsAsRegexes(topics []string) ([]*regexp.Regexp, error) {
	regexes := make([]*regexp.Regexp, len(topics))
	for i, topic := range topics {
		reg := compileTopicAsRegex(topic)
		if reg == nil {
			return nil, ierrors.Errorf("invalid topic in config: %s", topic)
		}
		regexes[i] = reg
	}

	return regexes, nil
}
