package restclient_test

// responsesValidateAgainst validates responses against an explicit .hresp path.
func (p *testFixture) responsesValidateAgainst(expectedPath string) *testFixture {
	p.require.NotNil(p.client)
	p.assert.NoError(p.client.ValidateResponses(expectedPath, p.responses...))

	return p
}
