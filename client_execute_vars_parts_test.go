package restclient_test

// responsesValidateAgainst validates responses against an explicit .hresp path.
func (p *parts) responsesValidateAgainst(expectedPath string) *parts {
	p.require.NotNil(p.client)
	p.assert.NoError(p.client.ValidateResponses(expectedPath, p.responses...))

	return p
}
