package cloudhub2

import "testing"

// TestTransitGatewayStatus_AWSTransitGatewayID pins the recovery of a bare AWS
// transit gateway id from status.tgwResource.
//
// The platform returns a console deep link there rather than an identifier, so an
// attribute named aws_transit_gateway_id was handing callers a URL — unusable for
// the obvious purpose of feeding it to the AWS provider. Every mock in this
// package fed a bare id, which is why nothing caught it; the production case
// below is copied from a live response.
func TestTransitGatewayStatus_AWSTransitGatewayID(t *testing.T) {
	tests := []struct {
		name        string
		tgwResource string
		want        string
	}{
		{
			name:        "console link returned by the platform yields the bare id",
			tgwResource: "https://console.aws.amazon.com/vpc/home?region=us-east-2#TransitGatewayDetails:transitGatewayId=tgw-0af6a0a1b5ae060b1",
			want:        "tgw-0af6a0a1b5ae060b1",
		},
		{
			name:        "a bare id passes through untouched",
			tgwResource: "tgw-0abc",
			want:        "tgw-0abc",
		},
		{
			name:        "empty stays empty rather than becoming a sentinel",
			tgwResource: "",
			want:        "",
		},
		{
			name:        "surrounding whitespace is trimmed",
			tgwResource: "  tgw-0abc  ",
			want:        "tgw-0abc",
		},
		{
			name:        "parameter in the query string rather than the fragment",
			tgwResource: "https://console.aws.amazon.com/vpc/home?transitGatewayId=tgw-0abc&region=us-east-2",
			want:        "tgw-0abc",
		},
		{
			name:        "trailing parameter after the id is not swallowed",
			tgwResource: "https://example.com/#transitGatewayId=tgw-0abc&foo=bar",
			want:        "tgw-0abc",
		},
		// The remaining cases are the safety net: an unrecognised value must fall
		// back to the platform's raw string. Returning empty would turn a cosmetic
		// problem into a silent data loss if AWS ever restyles the console link.
		{
			name:        "unrecognised format falls back to the raw value",
			tgwResource: "https://console.aws.amazon.com/vpc/home?region=us-east-2",
			want:        "https://console.aws.amazon.com/vpc/home?region=us-east-2",
		},
		{
			name:        "present but empty parameter falls back to the raw value",
			tgwResource: "https://example.com/#transitGatewayId=",
			want:        "https://example.com/#transitGatewayId=",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status := TransitGatewayStatus{TgwResource: tc.tgwResource}
			if got := status.AWSTransitGatewayID(); got != tc.want {
				t.Errorf("AWSTransitGatewayID() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestTransitGatewayStatus_AWSConsoleURL covers the other half of tgwResource.
// The Anypoint UI shows the id and a "View on AWS" link side by side, both from
// this one field, so extracting the id must not discard the link.
func TestTransitGatewayStatus_AWSConsoleURL(t *testing.T) {
	tests := []struct {
		name        string
		tgwResource string
		want        string
	}{
		{
			name:        "console link is returned in full",
			tgwResource: "https://console.aws.amazon.com/vpc/home?region=us-east-2#TransitGatewayDetails:transitGatewayId=tgw-0af6a0a1b5ae060b1",
			want:        "https://console.aws.amazon.com/vpc/home?region=us-east-2#TransitGatewayDetails:transitGatewayId=tgw-0af6a0a1b5ae060b1",
		},
		{
			name:        "plain http is still a link",
			tgwResource: "http://example.com/#transitGatewayId=tgw-0abc",
			want:        "http://example.com/#transitGatewayId=tgw-0abc",
		},
		// A bare id is an id, not a link. Echoing it into a URL-typed attribute
		// would hand callers something that looks like a link but is not.
		{
			name:        "a bare id yields no link",
			tgwResource: "tgw-0abc",
			want:        "",
		},
		{
			name:        "empty yields no link",
			tgwResource: "",
			want:        "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status := TransitGatewayStatus{TgwResource: tc.tgwResource}
			if got := status.AWSConsoleURL(); got != tc.want {
				t.Errorf("AWSConsoleURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestTransitGatewayStatus_IDAndURLAreComplementary states the invariant the two
// accessors exist to satisfy: for the console link the platform actually sends,
// neither accessor loses information — the id is usable on its own and the link
// survives intact.
func TestTransitGatewayStatus_IDAndURLAreComplementary(t *testing.T) {
	const link = "https://console.aws.amazon.com/vpc/home?region=us-east-2#TransitGatewayDetails:transitGatewayId=tgw-0af6a0a1b5ae060b1"
	status := TransitGatewayStatus{TgwResource: link}

	if id := status.AWSTransitGatewayID(); id != "tgw-0af6a0a1b5ae060b1" {
		t.Errorf("id = %q, want the bare identifier", id)
	}
	if url := status.AWSConsoleURL(); url != link {
		t.Errorf("url = %q, want the link unchanged", url)
	}
}
