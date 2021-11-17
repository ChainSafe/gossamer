package main

import (
	"fmt"
	"log"

	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v2"
)

type options struct {
	Namespace string   `short:"n" long:"namespace" description:"namespace that is prepended to all metrics" required:"true"`
	Tags      []string `short:"t" long:"tags" description:"tags that are added to all metrics"`
}

func main() {
	var opts options
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Panicf("%v", err)
	}
	yml, err := marshalYAML(opts)
	if err != nil {
		log.Panicf("%v", err)
	}
	fmt.Printf("%s", yml)
}

func marshalYAML(opts options) (yml []byte, err error) {
	var c conf
	err = yaml.Unmarshal([]byte(confYAML), &c)
	if err != nil {
		return
	}

	c.Instances[0].Namespace = opts.Namespace
	c.Instances[0].Tags = opts.Tags

	yml, err = yaml.Marshal(c)
	return
}

type instance struct {
	PrometheusURL      string   `yaml:"prometheus_url"`
	Namespace          string   `yaml:"namespace"`
	Metrics            []string `yaml:"metrics"`
	HealthServiceCheck bool     `yaml:"health_service_check"`
	Tags               []string `yaml:"tags,omitempty"`
}

type conf struct {
	InitConfig struct{}   `yaml:"init_config"`
	Instances  []instance `yaml:"instances"`
}

const confYAML = `
## All options defined here are available to all instances.
#
init_config:

    ## @param proxy - mapping - optional
    ## Set HTTP or HTTPS proxies for all instances. Use the "no_proxy" list
    ## to specify hosts that must bypass proxies.
    ##
    ## The SOCKS protocol is also supported like so:
    ##
    ##   socks5://user:pass@host:port
    ##
    ## Using the scheme "socks5" causes the DNS resolution to happen on the
    ## client, rather than on the proxy server. This is in line with "curl",
    ## which uses the scheme to decide whether to do the DNS resolution on
    ## the client or proxy. If you want to resolve the domains on the proxy
    ## server, use "socks5h" as the scheme.
    #
    # proxy:
    #   http: http://<PROXY_SERVER_FOR_HTTP>:<PORT>
    #   https: https://<PROXY_SERVER_FOR_HTTPS>:<PORT>
    #   no_proxy:
    #   - <HOSTNAME_1>
    #   - <HOSTNAME_2>

    ## @param skip_proxy - boolean - optional - default: false
    ## If set to "true", this makes the check bypass any proxy
    ## settings enabled and attempt to reach services directly.
    #
    # skip_proxy: false

    ## @param timeout - number - optional - default: 10
    ## The timeout for connecting to services.
    #
    # timeout: 10

    ## @param service - string - optional
    ## Attach the tag "service:<SERVICE>" to every metric, event, and service check emitted by this integration.
    ##
    ## Additionally, this sets the default "service" for every log source.
    #
    # service: <SERVICE>

## Every instance is scheduled independent of the others.
#
instances:

    ## @param prometheus_url - string - required
    ## The URL where your application metrics are exposed by Prometheus.
    #
  - prometheus_url: http://127.0.0.1:9876/metrics

    ## @param namespace - string - required
    ## The namespace to be prepended to all metrics.
    #
    namespace: gossamer.local.devnet

    ## @param metrics - (list of string or mapping) - required
    ## List of metrics to be fetched from the prometheus endpoint, if there's a
    ## value it'll be renamed. This list should contain at least one metric.
    #
    metrics:
      - gossamer_*
      - network_*
      - service_*
      - system_*

    ## @param prometheus_metrics_prefix - string - optional
    ## Removes a given <PREFIX> from exposed Prometheus metrics.
    #
    # prometheus_metrics_prefix: <PREFIX>_

    ## @param health_service_check - boolean - optional - default: true
    ## Send a service check reporting about the health of the Prometheus endpoint.
    ## The service check is named <NAMESPACE>.prometheus.health
    #
    health_service_check: true

    ## @param label_to_hostname - string - optional
    ## Override the hostname with the value of one label.
    #
    # label_to_hostname: <LABEL>

    ## @param label_joins - mapping - optional
    ## Allows targeting a metric to retrieve its label with a 1:1 mapping.
    #
    # label_joins:
    #   target_metric:
    #     label_to_match: <MATCHED_LABEL>
    #     labels_to_get:
    #     - <EXTRA_LABEL_1>
    #     - <EXTRA_LABEL_2>

    ## @param labels_mapper - mapping - optional
    ## The label mapper allows you to rename labels.
    ## Format is <LABEL_TO_RENAME>: <NEW_LABEL_NAME>
    #
    # labels_mapper:
    #   flavor: origin

    ## @param type_overrides - mapping - optional
    ## Override a type in the Prometheus payload or type an untyped metric (ignored by default).
    ## Supported <METRIC_TYPE> are "gauge", "counter", "histogram", and "summary".
    ## The "*" wildcard can be used to match multiple metric names.
    #
    # type_overrides:
    #   <METRIC_NAME>: <METRIC_TYPE>

    ## @param send_histograms_buckets - boolean - optional - default: true
    ## Set send_histograms_buckets to true to send the histograms bucket.
    #
    # send_histograms_buckets: true

    ## @param send_distribution_buckets - boolean - optional - default: false
    ## Set "send_distribution_buckets" to "true" to send histograms as Datadog distribution metrics.
    ##
    ## Learn more about distribution metrics: https://docs.datadoghq.com/developers/metrics/distributions/
    #
    # send_distribution_buckets: false

    ## @param send_monotonic_counter - boolean - optional - default: true
    ## Set send_monotonic_counter to true to send counters as monotonic counter.
    #
    # send_monotonic_counter: true

    ## @param send_distribution_counts_as_monotonic - boolean - optional - default: false
    ## If set to true, sends histograms and summary counters as monotonic counters (instead of gauges).
    #
    # send_distribution_counts_as_monotonic: false

    ## @param send_distribution_sums_as_monotonic - boolean - optional - default: false
    ## If set to true, sends histograms and summary sums as monotonic counters (instead of gauges).
    #
    # send_distribution_sums_as_monotonic: false

    ## @param exclude_labels - list of strings - optional
    ## A list of labels to be excluded
    #
    # exclude_labels:
    #   - timestamp

    ## @param bearer_token_auth - boolean - optional - default: false
    ## If set to true, adds a bearer token authentication header.
    ## Note: If bearer_token_path is not set, the default path is /var/run/secrets/kubernetes.io/serviceaccount/token.
    #
    # bearer_token_auth: false

    ## @param bearer_token_path - string - optional
    ## The path to a Kubernetes service account bearer token file. Make sure the file exists and is mounted correctly.
    ## Note: bearer_token_auth should be set to true to enable adding the token to HTTP headers for authentication.
    #
    # bearer_token_path: <TOKEN_PATH>

    ## @param ignore_metrics - list of strings - optional
    ## A list of metrics to ignore, use the "*" wildcard can be used to match multiple metric names.
    #
    # ignore_metrics:
    #   - <IGNORED_METRIC_NAME>
    #   - <PREFIX_*>
    #   - <*_SUFFIX>
    #   - <PREFIX_*_SUFFIX>
    #   - <*_SUBSTRING_*>

    ## @param ignore_metrics_by_labels - mapping - optional
    ## A mapping of labels where metrics with matching label key and values are ignored.
    ## Use the "*" wildcard to match all label values.
    #
    # ignore_metrics_by_labels:
    #   <KEY_1>:
    #   - <LABEL_1>
    #   - <LABEL_2>
    #   <KEY_2>:
    #   - '*'

    ## @param ignore_tags - list of strings - optional
    ## A list of regular expressions used to ignore tags added by autodiscovery and entries in the "tags" option.
    #
    # ignore_tags:
    #   - <FULL:TAG>
    #   - <TAG_PREFIX:.*>
    #   - <TAG_SUFFIX$>

    ## @param proxy - mapping - optional
    ## This overrides the "proxy" setting in "init_config".
    ##
    ## Set HTTP or HTTPS proxies for this instance. Use the "no_proxy" list
    ## to specify hosts that must bypass proxies.
    ##
    ## The SOCKS protocol is also supported, for example:
    ##
    ##   socks5://user:pass@host:port
    ##
    ## Using the scheme "socks5" causes the DNS resolution to happen on the
    ## client, rather than on the proxy server. This is in line with "curl",
    ## which uses the scheme to decide whether to do the DNS resolution on
    ## the client or proxy. If you want to resolve the domains on the proxy
    ## server, use "socks5h" as the scheme.
    #
    # proxy:
    #   http: http://<PROXY_SERVER_FOR_HTTP>:<PORT>
    #   https: https://<PROXY_SERVER_FOR_HTTPS>:<PORT>
    #   no_proxy:
    #   - <HOSTNAME_1>
    #   - <HOSTNAME_2>

    ## @param skip_proxy - boolean - optional - default: false
    ## This overrides the "skip_proxy" setting in "init_config".
    ##
    ## If set to "true", this makes the check bypass any proxy
    ## settings enabled and attempt to reach services directly.
    #
    # skip_proxy: false

    ## @param auth_type - string - optional - default: basic
    ## The type of authentication to use. The available types (and related options) are:
    ##
    ##   - basic
    ##     |__ username
    ##     |__ password
    ##     |__ use_legacy_auth_encoding
    ##   - digest
    ##     |__ username
    ##     |__ password
    ##   - ntlm
    ##     |__ ntlm_domain
    ##     |__ password
    ##   - kerberos
    ##     |__ kerberos_auth
    ##     |__ kerberos_cache
    ##     |__ kerberos_delegate
    ##     |__ kerberos_force_initiate
    ##     |__ kerberos_hostname
    ##     |__ kerberos_keytab
    ##     |__ kerberos_principal
    ##   - aws
    ##     |__ aws_region
    ##     |__ aws_host
    ##     |__ aws_service
    ##
    ## The "aws" auth type relies on boto3 to automatically gather AWS credentials, for example: from ".aws/credentials".
    ## Details: https://boto3.amazonaws.com/v1/documentation/api/latest/guide/configuration.html#configuring-credentials
    #
    # auth_type: basic

    ## @param use_legacy_auth_encoding - boolean - optional - default: true
    ## When "auth_type" is set to "basic", this determines whether to encode as "latin1" rather than "utf-8".
    #
    # use_legacy_auth_encoding: true

    ## @param username - string - optional
    ## The username to use if services are behind basic or digest auth.
    #
    # username: <USERNAME>

    ## @param password - string - optional
    ## The password to use if services are behind basic or NTLM auth.
    #
    # password: <PASSWORD>

    ## @param ntlm_domain - string - optional
    ## If your services use NTLM authentication, specify
    ## the domain used in the check. For NTLM Auth, append
    ## the username to domain, not as the "username" parameter.
    #
    # ntlm_domain: <NTLM_DOMAIN>\<USERNAME>

    ## @param kerberos_auth - string - optional - default: disabled
    ## If your services use Kerberos authentication, you can specify the Kerberos
    ## strategy to use between:
    ##
    ##   - required
    ##   - optional
    ##   - disabled
    ##
    ## See https://github.com/requests/requests-kerberos#mutual-authentication
    #
    # kerberos_auth: disabled

    ## @param kerberos_cache - string - optional
    ## Sets the KRB5CCNAME environment variable.
    ## It should point to a credential cache with a valid TGT.
    #
    # kerberos_cache: <KERBEROS_CACHE>

    ## @param kerberos_delegate - boolean - optional - default: false
    ## Set to "true" to enable Kerberos delegation of credentials to a server that requests delegation.
    ##
    ## See https://github.com/requests/requests-kerberos#delegation
    #
    # kerberos_delegate: false

    ## @param kerberos_force_initiate - boolean - optional - default: false
    ## Set to "true" to preemptively initiate the Kerberos GSS exchange and
    ## present a Kerberos ticket on the initial request (and all subsequent).
    ##
    ## See https://github.com/requests/requests-kerberos#preemptive-authentication
    #
    # kerberos_force_initiate: false

    ## @param kerberos_hostname - string - optional
    ## Override the hostname used for the Kerberos GSS exchange if its DNS name doesn't
    ## match its Kerberos hostname, for example: behind a content switch or load balancer.
    ##
    ## See https://github.com/requests/requests-kerberos#hostname-override
    #
    # kerberos_hostname: <KERBEROS_HOSTNAME>

    ## @param kerberos_principal - string - optional
    ## Set an explicit principal, to force Kerberos to look for a
    ## matching credential cache for the named user.
    ##
    ## See https://github.com/requests/requests-kerberos#explicit-principal
    #
    # kerberos_principal: <KERBEROS_PRINCIPAL>

    ## @param kerberos_keytab - string - optional
    ## Set the path to your Kerberos key tab file.
    #
    # kerberos_keytab: <KEYTAB_FILE_PATH>

    ## @param auth_token - mapping - optional
    ## This allows for the use of authentication information from dynamic sources.
    ## Both a reader and writer must be configured.
    ##
    ## The available readers are:
    ##
    ##   - type: file
    ##     path (required): The absolute path for the file to read from.
    ##     pattern: A regular expression pattern with a single capture group used to find the
    ##              token rather than using the entire file, for example: Your secret is (.+)
    ##
    ## The available writers are:
    ##
    ##   - type: header
    ##     name (required): The name of the field, for example: Authorization
    ##     value: The template value, for example "Bearer <TOKEN>". The default is: <TOKEN>
    ##     placeholder: The substring in "value" to replace by the token, defaults to: <TOKEN>
    #
    # auth_token:
    #   reader:
    #     type: <READER_TYPE>
    #     <OPTION_1>: <VALUE_1>
    #     <OPTION_2>: <VALUE_2>
    #   writer:
    #     type: <WRITER_TYPE>
    #     <OPTION_1>: <VALUE_1>
    #     <OPTION_2>: <VALUE_2>

    ## @param aws_region - string - optional
    ## If your services require AWS Signature Version 4 signing, set the region.
    ##
    ## See https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html
    #
    # aws_region: <AWS_REGION>

    ## @param aws_host - string - optional
    ## If your services require AWS Signature Version 4 signing, set the host.
    ##
    ## Note: This setting is not necessary for official integrations.
    ##
    ## See https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html
    #
    # aws_host: <AWS_HOST>

    ## @param aws_service - string - optional
    ## If your services require AWS Signature Version 4 signing, set the service code. For a list
    ## of available service codes, see https://docs.aws.amazon.com/general/latest/gr/rande.html
    ##
    ## Note: This setting is not necessary for official integrations.
    ##
    ## See https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html
    #
    # aws_service: <AWS_SERVICE>

    ## @param tls_verify - boolean - optional - default: true
    ## Instructs the check to validate the TLS certificate of services.
    #
    # tls_verify: true

    ## @param tls_use_host_header - boolean - optional - default: false
    ## If a "Host" header is set, this enables its use for SNI (matching against the TLS certificate CN or SAN).
    #
    # tls_use_host_header: false

    ## @param tls_ignore_warning - boolean - optional - default: false
    ## If "tls_verify" is disabled, security warnings are logged by the check.
    ## Disable those by setting "tls_ignore_warning" to true.
    ##
    ## Note: "tls_ignore_warning" set to true is currently only reliable if used by one instance of one integration.
    ## If enabled for multiple instances, spurious warnings might still appear even if "tls_ignore_warning" is set
    ## to true.
    #
    # tls_ignore_warning: false

    ## @param tls_cert - string - optional
    ## The path to a single file in PEM format containing a certificate as well as any
    ## number of CA certificates needed to establish the certificate's authenticity for
    ## use when connecting to services. It may also contain an unencrypted private key to use.
    #
    # tls_cert: <CERT_PATH>

    ## @param tls_private_key - string - optional
    ## The unencrypted private key to use for "tls_cert" when connecting to services. This is
    ## required if "tls_cert" is set and it does not already contain a private key.
    #
    # tls_private_key: <PRIVATE_KEY_PATH>

    ## @param tls_ca_cert - string - optional
    ## The path to a file of concatenated CA certificates in PEM format or a directory
    ## containing several CA certificates in PEM format. If a directory, the directory
    ## must have been processed using the c_rehash utility supplied with OpenSSL. See:
    ## https://www.openssl.org/docs/manmaster/man3/SSL_CTX_load_verify_locations.html
    #
    # tls_ca_cert: <CA_CERT_PATH>

    ## @param headers - mapping - optional
    ## The headers parameter allows you to send specific headers with every request.
    ## You can use it for explicitly specifying the host header or adding headers for
    ## authorization purposes.
    ##
    ## This overrides any default headers.
    #
    # headers:
    #   Host: <ALTERNATIVE_HOSTNAME>
    #   X-Auth-Token: <AUTH_TOKEN>

    ## @param extra_headers - mapping - optional
    ## Additional headers to send with every request.
    #
    # extra_headers:
    #   Host: <ALTERNATIVE_HOSTNAME>
    #   X-Auth-Token: <AUTH_TOKEN>

    ## @param timeout - number - optional - default: 10
    ## The timeout for accessing services.
    ##
    ## This overrides the "timeout" setting in "init_config".
    #
    # timeout: 10

    ## @param connect_timeout - number - optional
    ## The connect timeout for accessing services. Defaults to "timeout".
    #
    # connect_timeout: <CONNECT_TIMEOUT>

    ## @param read_timeout - number - optional
    ## The read timeout for accessing services. Defaults to "timeout".
    #
    # read_timeout: <READ_TIMEOUT>

    ## @param log_requests - boolean - optional - default: false
    ## Whether or not to debug log the HTTP(S) requests made, including the method and URL.
    #
    # log_requests: false

    ## @param persist_connections - boolean - optional - default: false
    ## Whether or not to persist cookies and use connection pooling for increased performance.
    #
    # persist_connections: false

    ## @param tags - list of strings - optional
    ## A list of tags to attach to every metric and service check emitted by this instance.
    ##
    ## Learn more about tagging at https://docs.datadoghq.com/tagging
    #
    tags:
    #   - <KEY_1>:<VALUE_1>
    #   - <KEY_2>:<VALUE_2>

    ## @param service - string - optional
    ## Attach the tag "service:<SERVICE>" to every metric, event, and service check emitted by this integration.
    ##
    ## Overrides any "service" defined in the "init_config" section.
    #
    # service: <SERVICE>

    ## @param min_collection_interval - number - optional - default: 15
    ## This changes the collection interval of the check. For more information, see:
    ## https://docs.datadoghq.com/developers/write_agent_check/#collection-interval
    #
    # min_collection_interval: 15

    ## @param empty_default_hostname - boolean - optional - default: false
    ## This forces the check to send metrics with no hostname.
    ##
    ## This is useful for cluster-level checks.
    #
    # empty_default_hostname: false
`
