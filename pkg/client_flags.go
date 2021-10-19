package pkg

import (
	"crypto/tls"
	"fmt"
	"github.com/urfave/cli/v2"
	"golang.org/x/net/http2"
	"io"
	"msg/pkg/go-workwx-develop"
	"net"
	"net/http"
	"os"
	"time"
)

const (
	flagCorpID            = "corpid"
	flagCorpSecret        = "corpsecret"
	flagAgentID           = "agentid"
	flagQyapiHostOverride = "qyapi-host-override"
	flagTLSKeyLogFile     = "tls-key-logfile"

	flagMessageType  = "message-type"
	flagSafe         = "safe"
	flagToUser       = "to-user"
	flagToUserShort  = "u"
	flagToParty      = "to-party"
	flagToPartyShort = "p"
	flagToTag        = "to-tag"
	flagToTagShort   = "t"
	flagToChat       = "to-chat"
	flagToChatShort  = "c"

	flagMediaID          = "media-id"
	flagThumbMediaID     = "thumb-media-id"
	flagDescription      = "desc"
	flagTitle            = "title"
	flagAuthor           = "author"
	flagURL              = "url"
	flagPicURL           = "pic-url"
	flagButtonText       = "button-text"
	flagSourceContentURL = "source-content-url"
	flagDigest           = "digest"

	flagMediaType = "media-type"
)

type CliOptions struct {
	CorpID            string
	CorpSecret        string
	AgentID           int64
	QYAPIHostOverride string
	TLSKeyLogFile     string
}

func mustGetConfig(c *cli.Context) *CliOptions {
	if !c.IsSet(flagCorpID) {
		panic("corpid must be set")
	}

	if !c.IsSet(flagCorpSecret) {
		panic("corpsecret must be set")
	}

	if !c.IsSet(flagAgentID) {
		panic("agentid must be set (for now; may later lift the restriction)")
	}

	return &CliOptions{
		CorpID:     c.String(flagCorpID),
		CorpSecret: c.String(flagCorpSecret),
		AgentID:    c.Int64(flagAgentID),

		QYAPIHostOverride: c.String(flagQyapiHostOverride),
		TLSKeyLogFile:     c.String(flagTLSKeyLogFile),
	}
}

//
// impl CliOptions
//

func (c *CliOptions) makeHTTPClient() *http.Client {
	if c.TLSKeyLogFile == "" {
		return http.DefaultClient
	}

	f, err := os.OpenFile(c.TLSKeyLogFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		fmt.Printf("can't open TLS key log file for writing: %+v\n", err)
		panic(err)
	}

	fmt.Fprintf(f, "# SSL/TLS secrets log file, generated by go\n")

	return &http.Client{
		Transport: newTransportWithKeyLog(f),
	}
}

func (c *CliOptions) makeWorkwxClient() *workwx.WorkWX {
	httpClient := c.makeHTTPClient()
	if c.QYAPIHostOverride != "" {
		// wtf think of a way to change this
		return workwx.New(c.CorpID,
			workwx.WithQYAPIHost(c.QYAPIHostOverride),
			workwx.WithHTTPClient(httpClient),
		)
	}
	return workwx.New(c.CorpID, workwx.WithHTTPClient(httpClient))
}

func (c *CliOptions) MakeWorkwxApp() *workwx.App {
	return c.makeWorkwxClient().WithApp(c.CorpSecret, c.AgentID)
}

// newTransportWithKeyLog initializes a HTTP Transport with KeyLogWriter
func newTransportWithKeyLog(keyLog io.Writer) *http.Transport {
	transport := &http.Transport{
		//nolint: gosec  // this transport is delibrately made to be a side channel
		TLSClientConfig: &tls.Config{KeyLogWriter: keyLog, InsecureSkipVerify: true},

		// Copy of http.DefaultTransport
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if err := http2.ConfigureTransport(transport); err != nil {
		panic(err)
	}
	return transport
}
