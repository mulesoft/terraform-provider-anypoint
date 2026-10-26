package exchange

// Exchange stores and returns GraphQL assets as type "graphql", while the provider
// (the anypoint_exchange_asset resource and the docs) calls the type "graphql-api".
// Without this mapping the documented filter `type = "graphql-api"` matched nothing
// and the data sources reported a different type than the resource.
const (
	graphqlUserType   = "graphql-api"
	graphqlStoredType = "graphql"
)

// storedAssetType maps a user-facing type filter to the value Exchange searches on.
func storedAssetType(t string) string {
	if t == graphqlUserType {
		return graphqlStoredType
	}
	return t
}

// userAssetType maps a type returned by Exchange to the user-facing name.
func userAssetType(t string) string {
	if t == graphqlStoredType {
		return graphqlUserType
	}
	return t
}
