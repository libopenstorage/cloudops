package containerservice

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
//
// Code generated by Microsoft (R) AutoRest Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"context"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/Azure/go-autorest/tracing"
	"net/http"
)

// ResolvePrivateLinkServiceIDClient is the the Container Service Client.
type ResolvePrivateLinkServiceIDClient struct {
	BaseClient
}

// NewResolvePrivateLinkServiceIDClient creates an instance of the ResolvePrivateLinkServiceIDClient client.
func NewResolvePrivateLinkServiceIDClient(subscriptionID string) ResolvePrivateLinkServiceIDClient {
	return NewResolvePrivateLinkServiceIDClientWithBaseURI(DefaultBaseURI, subscriptionID)
}

// NewResolvePrivateLinkServiceIDClientWithBaseURI creates an instance of the ResolvePrivateLinkServiceIDClient client
// using a custom endpoint.  Use this when interacting with an Azure cloud that uses a non-standard base URI (sovereign
// clouds, Azure stack).
func NewResolvePrivateLinkServiceIDClientWithBaseURI(baseURI string, subscriptionID string) ResolvePrivateLinkServiceIDClient {
	return ResolvePrivateLinkServiceIDClient{NewWithBaseURI(baseURI, subscriptionID)}
}

// POST sends the post request.
// Parameters:
// resourceGroupName - the name of the resource group.
// resourceName - the name of the managed cluster resource.
// parameters - parameters required in order to resolve a private link service ID.
func (client ResolvePrivateLinkServiceIDClient) POST(ctx context.Context, resourceGroupName string, resourceName string, parameters PrivateLinkResource) (result PrivateLinkResource, err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/ResolvePrivateLinkServiceIDClient.POST")
		defer func() {
			sc := -1
			if result.Response.Response != nil {
				sc = result.Response.Response.StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	if err := validation.Validate([]validation.Validation{
		{TargetValue: resourceGroupName,
			Constraints: []validation.Constraint{{Target: "resourceGroupName", Name: validation.MinLength, Rule: 1, Chain: nil}}},
		{TargetValue: resourceName,
			Constraints: []validation.Constraint{{Target: "resourceName", Name: validation.MaxLength, Rule: 63, Chain: nil},
				{Target: "resourceName", Name: validation.MinLength, Rule: 1, Chain: nil},
				{Target: "resourceName", Name: validation.Pattern, Rule: `^[a-zA-Z0-9]$|^[a-zA-Z0-9][-_a-zA-Z0-9]{0,61}[a-zA-Z0-9]$`, Chain: nil}}}}); err != nil {
		return result, validation.NewError("containerservice.ResolvePrivateLinkServiceIDClient", "POST", err.Error())
	}

	req, err := client.POSTPreparer(ctx, resourceGroupName, resourceName, parameters)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.ResolvePrivateLinkServiceIDClient", "POST", nil, "Failure preparing request")
		return
	}

	resp, err := client.POSTSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "containerservice.ResolvePrivateLinkServiceIDClient", "POST", resp, "Failure sending request")
		return
	}

	result, err = client.POSTResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.ResolvePrivateLinkServiceIDClient", "POST", resp, "Failure responding to request")
		return
	}

	return
}

// POSTPreparer prepares the POST request.
func (client ResolvePrivateLinkServiceIDClient) POSTPreparer(ctx context.Context, resourceGroupName string, resourceName string, parameters PrivateLinkResource) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"resourceName":      autorest.Encode("path", resourceName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2021-07-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	parameters.PrivateLinkServiceID = nil
	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPost(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/managedClusters/{resourceName}/resolvePrivateLinkServiceId", pathParameters),
		autorest.WithJSON(parameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// POSTSender sends the POST request. The method will close the
// http.Response Body if it receives an error.
func (client ResolvePrivateLinkServiceIDClient) POSTSender(req *http.Request) (*http.Response, error) {
	return client.Send(req, azure.DoRetryWithRegistration(client.Client))
}

// POSTResponder handles the response to the POST request. The method always
// closes the http.Response Body.
func (client ResolvePrivateLinkServiceIDClient) POSTResponder(resp *http.Response) (result PrivateLinkResource, err error) {
	err = autorest.Respond(
		resp,
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}
