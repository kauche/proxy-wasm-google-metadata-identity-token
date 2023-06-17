package internal

const (
	configKeyAudience                               = "audience"
	configKeyMetadataServerCluster                  = "metadata_server_cluster"
	configKeyTokenCacheDuration                     = "token_cache_duration"
	configKeyOriginalAuthorizationPropagationHeader = "original_authorization_propagation_header"

	defaultTokenCacheDuration = 60 * 30 // 30 min
)

type pluginConfiguration struct {
	Audience                               string `json:"audience"`
	MetadataServerCluster                  string `json:"metadata_server_cluster"`
	TokenCacheDuration                     uint64 `json:"token_cache_duration"`
	OriginalAuthorizationPropagationHeader string `json:"original_authorization_propagation_header"`
}
