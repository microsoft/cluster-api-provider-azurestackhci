package scope

import (
	"github.com/microsoft/moc/pkg/auth"
)

// ScopeInterface allows multiple scope types to be used for cloud services
type ScopeInterface interface {
	GetResourceGroup() string
	GetCloudAgentFqdn() string
	GetAuthorizer() auth.Authorizer
}
