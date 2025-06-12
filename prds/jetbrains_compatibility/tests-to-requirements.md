# Test Cases to HTTP Syntax Requirements Mapping

This document maps test cases from the `go-restclient` project to the relevant sections in `docs/http_syntax.md`.

**IMPORTANT NOTICE - DOCUMENT OUTDATED**: This mapping document contains references to test files and functions that no longer exist in the current codebase. The test structure has been consolidated since this document was created. 

**CURRENT STRUCTURE**: 
- Tests are now consolidated in `client_test.go`, `validator_test.go`, and `hresp_vars_test.go`
- Parser functionality is tested through client execution tests rather than separate parser unit tests
- See `test_coverage_mapping.md` for the corrected mapping of current tests to requirements

This document is preserved for historical reference but should not be used for current test planning.

## parser_test.go

### TestParseRequests_IgnoreEmptyBlocks

*   `parser_test.go/TestParseRequests_IgnoreEmptyBlocks/SCENARIO-LIB-028-001: File with only comments -> 249-256`
*   `parser_test.go/TestParseRequests_IgnoreEmptyBlocks/SCENARIO-LIB-028-002: File with only ### separators -> 200-210`
*   `parser_test.go/TestParseRequests_IgnoreEmptyBlocks/SCENARIO-LIB-028-003: File with comments and ### separators only -> 200-210, 249-256`
*   `parser_test.go/TestParseRequests_IgnoreEmptyBlocks/SCENARIO-LIB-028-004: Valid request, then separator, then only comments -> 130-193, 200-210, 249-256`
*   `parser_test.go/TestParseRequests_IgnoreEmptyBlocks/SCENARIO-LIB-028-005: Only comments, then separator, then valid request -> 130-193, 200-210, 249-256`
*   `parser_test.go/TestParseRequests_IgnoreEmptyBlocks/SCENARIO-LIB-028-006: Valid request, separator with comments, then another valid request -> 130-193, 200-210, 249-256`
*   `parser_test.go/TestParseRequests_IgnoreEmptyBlocks/Empty file content -> 130-193`
*   `parser_test.go/TestParseRequests_IgnoreEmptyBlocks/Single valid request no trailing newline -> 130-193`
*   `parser_test.go/TestParseRequests_IgnoreEmptyBlocks/File with only variable definitions -> 258-274`

### TestParseRequests_SeparatorComments

*   `parser_test.go/TestParseRequests_SeparatorComments/SCENARIO-LIB-027-001 & SCENARIO-LIB-027-004 combined -> 200-210, 249-256`
*   `parser_test.go/TestParseRequests_SeparatorComments/SCENARIO-LIB-027-003 style: Separator comment no newline before next request -> 200-210, 249-256`

### TestParseExpectedResponses_SeparatorComments

*   `parser_test.go/TestParseExpectedResponses_SeparatorComments/SCENARIO-LIB-027-002: Separator comment in response file -> 200-210, 249-256, 531-544`

### TestParseExpectedResponses_Simple

*   `parser_test.go/TestParseExpectedResponses_Simple/SCENARIO-LIB-007-001: Full valid response -> 531-544`
*   `parser_test.go/TestParseExpectedResponses_Simple/SCENARIO-LIB-007-002: Status line only -> 531-544`
*   `parser_test.go/TestParseExpectedResponses_Simple/SCENARIO-LIB-007-003: Status and headers only -> 531-544`
*   `parser_test.go/TestParseExpectedResponses_Simple/SCENARIO-LIB-007-004: Status and body only -> 531-544`
*   `parser_test.go/TestParseExpectedResponses_Simple/SCENARIO-LIB-007-005: Empty content -> 531-544`
*   `parser_test.go/TestParseExpectedResponses_Simple/SCENARIO-LIB-007-006: Malformed status line -> 531-544`
*   `parser_test.go/TestParseExpectedResponses_Simple/Multiple responses -> 200-210, 531-544`

### TestParseRequestFile_VariableScoping

*   `parser_test.go/TestParseRequestFile_VariableScoping -> 258-332`

### TestParseRequestFile_Imports

*   `parser_test.go/TestParseRequestFile_Imports/SCENARIO-IMPORT-001: Simple import - ignored -> Not Applicable (verifies absence of @import feature)`
*   `parser_test.go/TestParseRequestFile_Imports/SCENARIO-IMPORT-002: Nested import - ignored -> Not Applicable (verifies absence of @import feature)`
*   `parser_test.go/TestParseRequestFile_Imports/SCENARIO-IMPORT-003: Variable override - ignored -> Not Applicable (verifies absence of @import feature)`
*   `parser_test.go/TestParseRequestFile_Imports/SCENARIO-IMPORT-004: Circular import - ignored -> Not Applicable (verifies absence of @import feature)`
*   `parser_test.go/TestParseRequestFile_Imports/SCENARIO-IMPORT-005: Import not found - ignored -> Not Applicable (verifies absence of @import feature)`

### TestParserExternalFileDirectives

*   `parser_test.go/TestParserExternalFileDirectives/parse_external_file_with_encoding_directive -> 407-416`
*   `parser_test.go/TestParserExternalFileDirectives/parse_external_file_with_invalid_encoding_directive -> 407-416`
*   `parser_test.go/TestParserExternalFileDirectives/parse_external_file_without_encoding_directive -> 383-394`
*   `parser_test.go/TestParserExternalFileDirectives/parse_external_file_with_variables_and_encoding -> 396-405, 407-416`


## client_cookies_redirects_test.go

### TestCookieJarHandling

*   `client_cookies_redirects_test.go/TestCookieJarHandling -> 500-518, 598-600`

### TestRedirectHandling

*   `client_cookies_redirects_test.go/TestRedirectHandling -> 500-518, 602-606`


## client_execute_config_test.go

### TestExecuteFile_WithBaseURL

*   `client_execute_config_test.go/TestExecuteFile_WithBaseURL -> Not Applicable (test commented out)`

### TestExecuteFile_WithDefaultHeaders

*   `client_execute_config_test.go/TestExecuteFile_WithDefaultHeaders -> Not Applicable (test commented out)`

### TestMinimalInConfig

*   `client_execute_config_test.go/TestMinimalInConfig -> Not Applicable (placeholder test)`


## client_execute_core_test.go

### TestExecuteFile_SingleRequest

*   `client_execute_core_test.go/TestExecuteFile_SingleRequest -> 172-198, 212-220`

### TestExecuteFile_MultipleRequests

*   `client_execute_core_test.go/TestExecuteFile_MultipleRequests -> 172-241, 200-210, 402-420, 531-544`

### TestExecuteFile_RequestWithError

*   `client_execute_core_test.go/TestExecuteFile_RequestWithError -> 172-198, 200-210`

### TestExecuteFile_ParseError

*   `client_execute_core_test.go/TestExecuteFile_ParseError -> Not Applicable (tests parsing failure of invalid file)`

### TestExecuteFile_NoRequestsInFile

*   `client_execute_core_test.go/TestExecuteFile_NoRequestsInFile -> 243-256`

### TestExecuteFile_ValidThenInvalidSyntax

*   `client_execute_core_test.go/TestExecuteFile_ValidThenInvalidSyntax -> 172-198, 200-210`

### TestExecuteFile_MultipleErrors

*   `client_execute_core_test.go/TestExecuteFile_MultipleErrors -> 172-198, 200-210`

### TestExecuteFile_CapturesResponseHeaders

*   `client_execute_core_test.go/TestExecuteFile_CapturesResponseHeaders -> 531-544`

### TestExecuteFile_SimpleGetHTTP

*   `client_execute_core_test.go/TestExecuteFile_SimpleGetHTTP -> 172-198, 212-220`

### TestExecuteFile_MultipleRequests_GreaterThanTwo

*   `client_execute_core_test.go/TestExecuteFile_MultipleRequests_GreaterThanTwo -> 172-241, 200-210, 402-420, 531-544`


## client_execute_edgecases_test.go

### TestExecuteFile_InvalidMethodInFile

*   `client_execute_edgecases_test.go/TestExecuteFile_InvalidMethodInFile -> 212-220 (verifies handling of methods not conforming to this section)`

### TestExecuteFile_IgnoreEmptyBlocks_Client

*   `client_execute_edgecases_test.go/TestExecuteFile_IgnoreEmptyBlocks_Client/SCENARIO-LIB-028-004: Valid request, then separator, then only comments -> 172-198, 200-210, 243-256`
*   `client_execute_edgecases_test.go/TestExecuteFile_IgnoreEmptyBlocks_Client/SCENARIO-LIB-028-005: Only comments, then separator, then valid request -> 172-198, 200-210, 243-256`
*   `client_execute_edgecases_test.go/TestExecuteFile_IgnoreEmptyBlocks_Client/SCENARIO-LIB-028-006: Valid request, separator with comments, then another valid request -> 172-198, 200-210, 243-256`
*   `client_execute_edgecases_test.go/TestExecuteFile_IgnoreEmptyBlocks_Client/File with only variable definitions - ExecuteFile -> 258-274 (verifies error if only vars, no requests)`


## client_execute_external_file_test.go

### TestClientExecuteFileWithEncoding

*   `client_execute_external_file_test.go/TestClientExecuteFileWithEncoding/Latin-1_encoded_file -> 407-416`
*   `client_execute_external_file_test.go/TestClientExecuteFileWithEncoding/CP1252_(Windows-1252)_encoded_file -> 407-416`
*   `client_execute_external_file_test.go/TestClientExecuteFileWithEncoding/ASCII_encoded_file -> 407-416`
*   `client_execute_external_file_test.go/TestClientExecuteFileWithEncoding/UTF-8_encoded_file_(no_BOM) -> 407-416`
*   `client_execute_external_file_test.go/TestClientExecuteFileWithEncoding/UTF-8_encoded_file_with_BOM -> 407-416`
*   `client_execute_external_file_test.go/TestClientExecuteFileWithEncoding/Unsupported_encoding_specified -> 407-416`
*   `client_execute_external_file_test.go/TestClientExecuteFileWithEncoding/Encoding_specified_but_file_not_found -> 407-416`
*   `client_execute_external_file_test.go/TestClientExecuteFileWithEncoding/No_encoding_specified_defaults_to_UTF-8 -> 383-394`
*   `client_execute_external_file_test.go/TestClientExecuteFileWithEncoding/External_file_with_variables_and_encoding -> 396-405, 407-416`

### TestExecuteFile_ExternalFileWithEncoding

*   `client_execute_external_file_test.go/TestExecuteFile_ExternalFileWithEncoding/Valid_Latin-1_encoded_file -> 407-416`
*   `client_execute_external_file_test.go/TestExecuteFile_ExternalFileWithEncoding/Valid_CP1252_encoded_file -> 407-416`
*   `client_execute_external_file_test.go/TestExecuteFile_ExternalFileWithEncoding/Valid_ASCII_encoded_file -> 407-416`
*   `client_execute_external_file_test.go/TestExecuteFile_ExternalFileWithEncoding/Valid_UTF-8_encoded_file -> 407-416`
*   `client_execute_external_file_test.go/TestExecuteFile_ExternalFileWithEncoding/Unsupported_encoding -> 407-416`
*   `client_execute_external_file_test.go/TestExecuteFile_ExternalFileWithEncoding/File_not_found -> 407-416`


## client_execute_inplace_vars_test.go

### TestExecuteFile_InPlace_SimpleVariableInURL
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_SimpleVariableInURL -> 212-220, 258-274`

### TestExecuteFile_InPlace_VariableInHeader
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableInHeader -> 222-241, 258-274`

### TestExecuteFile_InPlace_VariableInBody
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableInBody -> 258-274, 402-420`

### TestExecuteFile_InPlace_VariableDefinedByAnotherVariable
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByAnotherVariable -> 258-274`

### TestExecuteFile_InPlace_VariablePrecedenceOverEnvironment
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariablePrecedenceOverEnvironment -> 258-332`

### TestExecuteFile_InPlace_VariableInCustomHeader
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableInCustomHeader -> 222-241, 258-274`

### TestExecuteFile_InPlace_VariableSubstitutionInBody
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableSubstitutionInBody -> 258-274, 402-420`

### TestExecuteFile_InPlace_VariableDefinedBySystemVariable
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedBySystemVariable -> 258-274, 333-360`

### TestExecuteFile_InPlace_VariableDefinedByOsEnvVariable
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByOsEnvVariable -> 258-274, 275-332`

### TestExecuteFile_InPlace_VariableInAuthHeader
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableInAuthHeader -> 222-241, 258-274, 578-596`

### TestExecuteFile_InPlace_VariableInJsonRequestBody
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableInJsonRequestBody -> 258-274, 402-420`

### TestExecuteFile_InPlace_VariableDefinedByAnotherInPlaceVariable
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByAnotherInPlaceVariable -> 258-274`

### TestExecuteFile_InPlace_VariableDefinedByDotEnvOsVariable
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByDotEnvOsVariable -> 258-274, 275-332`

### TestExecuteFile_InPlace_Malformed_NameOnlyNoEqualsNoValue
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_Malformed_NameOnlyNoEqualsNoValue -> 258-274 (verifies error on malformed syntax)`

### TestExecuteFile_InPlace_Malformed_NoNameEqualsValue
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_Malformed_NoNameEqualsValue -> 258-274 (verifies error on malformed syntax)`

### TestExecuteFile_InPlace_VariableDefinedByDotEnvSystemVariable
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByDotEnvSystemVariable -> 258-274, 275-332, 333-360`

### TestExecuteFile_InPlace_VariableDefinedByRandomInt
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByRandomInt -> 258-274, 333-360`


## client_execute_system_vars_test.go

### TestExecuteFile_WithGuidSystemVariable
*   `client_execute_system_vars_test.go/TestExecuteFile_WithGuidSystemVariable -> 333-338`

### TestExecuteFile_WithIsoTimestampSystemVariable
*   `client_execute_system_vars_test.go/TestExecuteFile_WithIsoTimestampSystemVariable -> 333-336, 340`

### TestExecuteFile_WithDatetimeSystemVariables
*   `client_execute_system_vars_test.go/TestExecuteFile_WithDatetimeSystemVariables -> 333-336, 341-345`

### TestExecuteFile_WithTimestampSystemVariable
*   `client_execute_system_vars_test.go/TestExecuteFile_WithTimestampSystemVariable -> 333-336, 339`

### TestExecuteFile_WithRandomIntSystemVariable
*   `client_execute_system_vars_test.go/TestExecuteFile_WithRandomIntSystemVariable/valid min max args -> 333-336, 347-350`
*   `client_execute_system_vars_test.go/TestExecuteFile_WithRandomIntSystemVariable/no args -> 333-336, 347-350`
*   `client_execute_system_vars_test.go/TestExecuteFile_WithRandomIntSystemVariable/swapped min max args -> 333-336, 347-350 (verifies error/fallback for invalid range)`
*   `client_execute_system_vars_test.go/TestExecuteFile_WithRandomIntSystemVariable/malformed args -> 333-336, 347-350 (verifies error/fallback for malformed args)`


## client_execute_vars_test.go

### TestRandomStringFromCharset
*   `client_execute_vars_test.go/TestRandomStringFromCharset -> N/A (helper function test)`

### TestSubstituteDynamicSystemVariables_EnvVars
*   `client_execute_vars_test.go/TestSubstituteDynamicSystemVariables_EnvVars -> 275-332, 343`

### TestExecuteFile_WithCustomVariables
*   `client_execute_vars_test.go/TestExecuteFile_WithCustomVariables -> 258-274`

### TestExecuteFile_WithProcessEnvSystemVariable
*   `client_execute_vars_test.go/TestExecuteFile_WithProcessEnvSystemVariable -> 275-332, 344`

### TestExecuteFile_WithDotEnvSystemVariable
*   `client_execute_vars_test.go/TestExecuteFile_WithDotEnvSystemVariable -> 275-332, 345`

### TestExecuteFile_WithProgrammaticVariables
*   `client_execute_vars_test.go/TestExecuteFile_WithProgrammaticVariables -> 362-370`

### TestExecuteFile_WithLocalDatetimeSystemVariable
*   `client_execute_vars_test.go/TestExecuteFile_WithLocalDatetimeSystemVariable -> 333-336, 342`

### TestExecuteFile_VariableFunctionConsistency
*   `client_execute_vars_test.go/TestExecuteFile_VariableFunctionConsistency -> 258-370 (covers interplay and precedence of all variable types)`

### TestExecuteFile_WithHttpClientEnvJson
*   `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson/SCENARIO-LIB-010-001: Basic env substitution from http-client.env.json -> 275-289`
*   `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson/SCENARIO-LIB-010-002: Private env overrides public env -> 275-297 (precedence)`
*   `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson/SCENARIO-LIB-010-003: In-place file var overrides private env -> 258-274, 291-297 (precedence)`
*   `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson/SCENARIO-LIB-010-004: Non-existent env file does not cause error -> 277-297 (behavior on missing files)`
*   `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson/SCENARIO-LIB-010-005: Malformed public env JSON -> 277-289 (error handling)`
*   `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson/SCENARIO-LIB-010-006: Malformed private env JSON -> 291-297 (error handling)`
*   `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson/SCENARIO-LIB-010-007: Environment selection via client option -> 306-317`
*   `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson/SCENARIO-LIB-010-008: Public env for selected environment -> 277-289, 306-317`
*   `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson/SCENARIO-LIB-010-009: Private env for selected environment overrides public -> 277-297, 306-317 (precedence)`
*   `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson/SCENARIO-LIB-010-010: All variables from default and selected env are available -> 275-332 (scoping and merging)`
*   `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson/SCENARIO-LIB-010-011: Selected env var overrides default env var -> 275-332 (precedence with selection)`
*   `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson/SCENARIO-LIB-010-012: Default private env var overrides default public env var -> 275-297 (precedence)`
*   `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson/SCENARIO-LIB-010-013: Selected private env var overrides selected public env var -> 275-297, 306-317 (precedence)`
*   `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson/SCENARIO-LIB-010-014: Selected private env var overrides default private env var -> 275-297, 306-317 (precedence)`

### TestExecuteFile_WithExtendedRandomSystemVariables
*   `client_execute_vars_test.go/TestExecuteFile_WithExtendedRandomSystemVariables -> 333-336, 352-360`


## client_init_test.go

### TestNewClient
*   `client_init_test.go/TestNewClient -> N/A (client initialization test)`

### TestNewClient_WithOptions
*   `client_init_test.go/TestNewClient_WithOptions -> N/A (client initialization with options test)`


## client_test_helpers_test.go

### TestCreateTestFileFromTemplate_DebugOutput
*   `client_test_helpers_test.go/TestCreateTestFileFromTemplate_DebugOutput -> N/A (helper function test)`


## hresp_vars_test.go

### TestExtractHrespDefines
*   `hresp_vars_test.go/TestExtractHrespDefines -> 258-274 (variable definition syntax)`

### TestResolveAndSubstitute
*   `hresp_vars_test.go/TestResolveAndSubstitute/file variable substitution -> 258-274`
*   `hresp_vars_test.go/TestResolveAndSubstitute/programmatic variable substitution -> 362-370`
*   `hresp_vars_test.go/TestResolveAndSubstitute/fallback value used -> 258-274 (general variable usage with fallback)`
*   `hresp_vars_test.go/TestResolveAndSubstitute/fallback value not used -> 258-274`
*   `hresp_vars_test.go/TestResolveAndSubstitute/programmatic var overrides file var -> 258-274, 362-370 (precedence)`
*   `hresp_vars_test.go/TestResolveAndSubstitute/file var overrides client env var -> 258-274, 275-332 (precedence)`
*   `hresp_vars_test.go/TestResolveAndSubstitute/programmatic var overrides client env var -> 275-332, 362-370 (precedence)`
*   `hresp_vars_test.go/TestResolveAndSubstitute/$uuid system variable -> 333-336, 339`
*   `hresp_vars_test.go/TestResolveAndSubstitute/$timestamp system variable -> 333-336, 338`
*   `hresp_vars_test.go/TestResolveAndSubstitute/$randomInt system variable -> 333-336, 347-350`
*   `hresp_vars_test.go/TestResolveAndSubstitute/$processEnvVariable system variable -> 275-332, 344`
*   `hresp_vars_test.go/TestResolveAndSubstitute/$dotenv system variable -> 275-332, 345`
*   `hresp_vars_test.go/TestResolveAndSubstitute/mixed custom (programmatic) and system -> 258-370 (interplay)`
*   `hresp_vars_test.go/TestResolveAndSubstitute/fallback providing a system variable to be resolved -> 333-360 (system var in fallback)`
*   `hresp_vars_test.go/TestResolveAndSubstitute/unresolved variable without fallback -> 258-274 (handling of undefined variables)`
*   `hresp_vars_test.go/TestResolveAndSubstitute/no variables in content -> N/A (no variables)`
*   `hresp_vars_test.go/TestResolveAndSubstitute/spaces around variable and fallback pipe (prog vars) -> 362-370 (syntax robustness)`
*   `hresp_vars_test.go/TestResolveAndSubstitute/spaces around variable, fallback used -> 258-274 (syntax robustness)`
*   `hresp_vars_test.go/TestResolveAndSubstitute/numeric programmatic var -> 362-370 (data types)`
*   `hresp_vars_test.go/TestResolveAndSubstitute/boolean programmatic var -> 362-370 (data types)`


## parser_authentication_test.go

### TestBasicAuthHeader
*   `parser_authentication_test.go/TestBasicAuthHeader -> 371-384 (Basic Auth via Header)`

### TestBasicAuthURL
*   `parser_authentication_test.go/TestBasicAuthURL -> 371-384 (Basic Auth via URL)`

### TestBearerTokenAuth
*   `parser_authentication_test.go/TestBearerTokenAuth -> 385-391 (Bearer Token)`

### TestOAuthFlowWithRequestReferences
*   `parser_authentication_test.go/TestOAuthFlowWithRequestReferences -> 392-418 (OAuth 2.0 Flow), 201-214 (Request References for token)`


## parser_environment_vars_test.go

### TestParseRequestFile_EnvironmentVariables
*   `parser_environment_vars_test.go/TestParseRequestFile_EnvironmentVariables -> 275-332 (Environment Variables, including default values)`

### TestParseRequestFile_VariableDefinitions
*   `parser_environment_vars_test.go/TestParseRequestFile_VariableDefinitions -> 258-274 (File Variables)`


## parser_request_body_test.go

### TestJsonRequestBodies
*   `parser_request_body_test.go/TestJsonRequestBodies -> 138-147 (JSON Body)`

### TestFileBasedBodies
*   `parser_request_body_test.go/TestFileBasedBodies -> 148-155 (File Input for Body)`

### TestFormUrlEncodedBodies
*   `parser_request_body_test.go/TestFormUrlEncodedBodies -> 156-163 (Form URL Encoded Body)`

### TestMultipartFormDataBodies
*   `parser_request_body_test.go/TestMultipartFormDataBodies -> 164-181 (Multipart Form Data - Text Fields)`

### TestFileUploadBodies
*   `parser_request_body_test.go/TestFileUploadBodies -> 164-181 (Multipart Form Data - File Uploads)`


## parser_request_settings_test.go

### TestNameDirective
*   `parser_request_settings_test.go/TestNameDirective -> 182-187 (@name)`

### TestNoRedirectDirective
*   `parser_request_settings_test.go/TestNoRedirectDirective -> 188-191 (@no-redirect)`

### TestNoCookieJarDirective
*   `parser_request_settings_test.go/TestNoCookieJarDirective -> 192-195 (@no-cookie-jar)`

### TestTimeoutDirective
*   `parser_request_settings_test.go/TestTimeoutDirective -> 196-200 (@timeout)`


## parser_request_structure_test.go

### TestParserRequestNaming
*   `parser_request_structure_test.go/TestParserRequestNaming -> 97-109 (Request Naming with ###)`

### TestParserCommentStyles
*   `parser_request_structure_test.go/TestParserCommentStyles -> 110-119 (Comment Styles # and //)`


## parser_response_validation_test.go

### TestParseResponseValidationPlaceholder_Any
*   `parser_response_validation_test.go/TestParseResponseValidationPlaceholder_Any -> 237-240 ({{$any}} placeholder)`

### TestParseResponseValidationPlaceholder_Regexp
*   `parser_response_validation_test.go/TestParseResponseValidationPlaceholder_Regexp -> 241-244 ({{$regexp}} placeholder)`

### TestParseResponseValidationPlaceholder_AnyGuid
*   `parser_response_validation_test.go/TestParseResponseValidationPlaceholder_AnyGuid -> 245-248 ({{$anyGuid}}/{{$anyUuid}} placeholder)`

### TestParseResponseValidationPlaceholder_AnyTimestamp
*   `parser_response_validation_test.go/TestParseResponseValidationPlaceholder_AnyTimestamp -> 249-252 ({{$anyTimestamp}} placeholder)`

### TestParseResponseValidationPlaceholder_AnyDatetime
*   `parser_response_validation_test.go/TestParseResponseValidationPlaceholder_AnyDatetime -> 253-257 ({{$anyDatetime 'format'}} placeholder)`

### TestParseChainedRequests
*   `parser_response_validation_test.go/TestParseChainedRequests -> 201-214 (Request References/Chaining)`


## parser_system_vars_test.go

### TestParseRequestFile_BasicSystemVariables
*   `parser_system_vars_test.go/TestParseRequestFile_BasicSystemVariables#UUID-GUID -> 334-337 (UUID/GUID System Variables)`
*   `parser_system_vars_test.go/TestParseRequestFile_BasicSystemVariables#Timestamps -> 338-342 (Timestamp System Variables)`
*   `parser_system_vars_test.go/TestParseRequestFile_BasicSystemVariables#Datetimes -> 343-348 (Datetime System Variables)`

### TestParseRequestFile_RandomSystemVariables
*   `parser_system_vars_test.go/TestParseRequestFile_RandomSystemVariables#RandomInt -> 349-354 (Random Integer System Variables)`
*   `parser_system_vars_test.go/TestParseRequestFile_RandomSystemVariables#RandomFloat -> 349-354 (Random Float System Variables)`
*   `parser_system_vars_test.go/TestParseRequestFile_RandomSystemVariables#RandomString -> 349-354 (Random String System Variables)`

### TestParseRequestFile_EnvironmentAccess
*   `parser_system_vars_test.go/TestParseRequestFile_EnvironmentAccess#processEnv -> 355-360 (System Env Var Access - $processEnv)`
*   `parser_system_vars_test.go/TestParseRequestFile_EnvironmentAccess#env -> 355-360 (System Env Var Access - $env)`


## validator_body_test.go

### TestValidateResponses_Body_ExactMatch
*   `validator_body_test.go/TestValidateResponses_Body_ExactMatch -> 221-227 (Expected Response Body - Exact Match from .hresp)`

### TestValidateResponses_BodyContains
*   `validator_body_test.go/TestValidateResponses_BodyContains -> 221-227 (Expected Response Body - .hresp interaction with BodyContains logic)`
    *   _Note: This test verifies that the programmatic `BodyContains` logic behaves correctly (falls back to exact match) when expected responses are loaded from `.hresp` files, which define exact bodies._

### TestValidateResponses_BodyNotContains
*   `validator_body_test.go/TestValidateResponses_BodyNotContains -> 221-227 (Expected Response Body - .hresp interaction with BodyNotContains logic)`
    *   _Note: This test verifies that the programmatic `BodyNotContains` logic behaves correctly (falls back to exact match) when expected responses are loaded from `.hresp` files, which define exact bodies._


## validator_general_test.go

### TestValidateResponses_WithSampleFile
Validates responses against a fully specified `.hresp` file (`testdata/http_response_files/sample1.http`).
*   `validator_general_test.go/TestValidateResponses_WithSampleFile/"perfect match with sample1.http" -> 212-215 (Overall .hresp), 216-220 (Status), 221-227 (Body), 228-236 (Headers)`
*   `validator_general_test.go/TestValidateResponses_WithSampleFile/"status code mismatch" -> 216-220 (Expected Response Status Code)`
*   `validator_general_test.go/TestValidateResponses_WithSampleFile/"status string mismatch" -> 216-220 (Expected Response Status String)`
*   `validator_general_test.go/TestValidateResponses_WithSampleFile/"header value mismatch for Content-Type" -> 228-236 (Expected Response Headers Validation)`
*   `validator_general_test.go/TestValidateResponses_WithSampleFile/"missing expected header Date" -> 228-236 (Expected Response Headers Validation)`
*   `validator_general_test.go/TestValidateResponses_WithSampleFile/"body mismatch" -> 221-227 (Expected Response Body - Exact Match from .hresp)`
*   `validator_general_test.go/TestValidateResponses_WithSampleFile/"BodyContains logic not triggered by exact file match (positive case)" -> 221-227 (Expected Response Body - .hresp interaction with programmatic contains logic)`
    *   _Note: Confirms .hresp exact body match takes precedence over programmatic `BodyContains` logic._
*   `validator_general_test.go/TestValidateResponses_WithSampleFile/"BodyContains logic not triggered, exact body mismatch from file" -> 221-227 (Expected Response Body - .hresp interaction with programmatic contains logic)`
    *   _Note: Confirms .hresp exact body match takes precedence over programmatic `BodyContains` logic._

### TestValidateResponses_PartialExpected
Validates responses against `.hresp` files that provide only partial expectations.
*   `validator_general_test.go/TestValidateResponses_PartialExpected/"SCENARIO-LIB-009-005 Equiv: Expected file has only status code - match" -> 216-220 (Expected Response Status Code - with partial .hresp)`
*   `validator_general_test.go/TestValidateResponses_PartialExpected/"SCENARIO-LIB-009-005 Corrected: File has status code and empty body - actual matches" -> 216-220 (Status), 221-227 (Body - implied/explicit empty in partial .hresp)`
*   `validator_general_test.go/TestValidateResponses_PartialExpected/"SCENARIO-LIB-009-005-003 Corrected: File has status code and empty body - actual body mismatch" -> 216-220 (Status), 221-227 (Body - implied/explicit empty in partial .hresp)`
*   `validator_general_test.go/TestValidateResponses_PartialExpected/"SCENARIO-LIB-009-005-004 Equiv: Expected file has only status code - status code mismatch" -> 216-220 (Expected Response Status Code - with partial .hresp)`
*   `validator_general_test.go/TestValidateResponses_PartialExpected/"SCENARIO-LIB-009-006 Equiv: Expected file has only specific headers (and status, empty body) - match" -> 216-220 (Status), 228-236 (Headers - with partial .hresp), 221-227 (Body - implied/explicit empty in partial .hresp)`
*   `validator_general_test.go/TestValidateResponses_PartialExpected/"SCENARIO-LIB-009-006-002 Equiv: Expected file has only specific headers - header value mismatch" -> 228-236 (Headers - with partial .hresp)`
*   `validator_general_test.go/TestValidateResponses_PartialExpected/"SCENARIO-LIB-009-006-003 Equiv: Expected file has only specific headers - header missing in actual" -> 228-236 (Headers - with partial .hresp)`


## validator_headers_test.go

### TestValidateResponses_Headers
Validates response headers against `.hresp` file specifications.
*   `validator_headers_test.go/TestValidateResponses_Headers/"matching headers" -> 228-236 (Expected Response Headers Validation)`
*   `validator_headers_test.go/TestValidateResponses_Headers/"mismatching header value" -> 228-236 (Expected Response Headers Validation - Value Mismatch)`
*   `validator_headers_test.go/TestValidateResponses_Headers/"missing expected header" -> 228-236 (Expected Response Headers Validation - Missing Header)`
*   `validator_headers_test.go/TestValidateResponses_Headers/"extra actual header (should be ignored)" -> 228-236 (Expected Response Headers Validation - Extra Actual Header Ignored)`
*   `validator_headers_test.go/TestValidateResponses_Headers/"matching multi-value headers (order preserved)" -> 228-236 (Expected Response Headers Validation - Multi-value Match)`
*   `validator_headers_test.go/TestValidateResponses_Headers/"mismatching multi-value headers (different order)" -> 228-236 (Expected Response Headers Validation - Multi-value Order Mismatch)`
*   `validator_headers_test.go/TestValidateResponses_Headers/"mismatching multi-value headers (different value)" -> 228-236 (Expected Response Headers Validation - Multi-value Different Value)`
*   `validator_headers_test.go/TestValidateResponses_Headers/"subset of multi-value headers (actual has more values)" -> 228-236 (Expected Response Headers Validation - Multi-value Subset)`
*   `validator_headers_test.go/TestValidateResponses_Headers/"case-insensitive header key matching" -> 228-236 (Expected Response Headers Validation - Case-Insensitive Key Match)`

### TestValidateResponses_HeadersContain
*   `validator_headers_test.go/TestValidateResponses_HeadersContain -> 228-236 (Expected Response Headers - .hresp interaction with HeadersContain logic)`
    *   _Note: This test verifies that programmatic `HeadersContain` logic behaves correctly (falls back to exact header set match) when expected responses are loaded from `.hresp` files._


## client_cookies_redirects_test.go

### TestCookieJarHandling
Tests the client's cookie jar functionality.
*   `client_cookies_redirects_test.go/TestCookieJarHandling/"with cookie jar (default)" -> 104-109 (Request Settings - @no-cookie-jar)`
    *   _Note: Verifies default behavior (cookie jar enabled) when `@no-cookie-jar` is NOT present._
*   `client_cookies_redirects_test.go/TestCookieJarHandling/"without cookie jar (@no-cookie-jar directive)" -> 104-109 (Request Settings - @no-cookie-jar)`

### TestRedirectHandling
Tests the client's redirect handling functionality.
*   `client_cookies_redirects_test.go/TestRedirectHandling/"with redirect following (default)" -> 97-102 (Request Settings - @no-redirect)`
    *   _Note: Verifies default behavior (redirects followed) when `@no-redirect` is NOT present._
*   `client_cookies_redirects_test.go/TestRedirectHandling/"without redirect following (@no-redirect directive)" -> 97-102 (Request Settings - @no-redirect)`


## client_execute_config_test.go
_Note: The primary test functions in this file, `TestExecuteFile_WithBaseURL` (related to client base URL configuration and its interaction with request lines) and `TestExecuteFile_WithDefaultHeaders` (related to client default headers and their interaction with request headers), are currently commented out. The active test `TestMinimalInConfig` is a placeholder and does not map to specific syntax features._

*   `client_execute_config_test.go/TestExecuteFile_WithBaseURL` (Commented Out) -> Potentially 38-46 (Request Line - interaction with client's BaseURL)
*   `client_execute_config_test.go/TestExecuteFile_WithDefaultHeaders` (Commented Out) -> Potentially 48-56 (Request Headers - interaction with client's Default Headers)


## client_execute_core_test.go

### TestExecuteFile_SingleRequest
Tests executing a file with a single, simple GET request.
*   `client_execute_core_test.go/TestExecuteFile_SingleRequest -> 38-64 (Basic Request Structure)`

### TestExecuteFile_MultipleRequests
Tests executing a file with multiple requests (GET and POST), including headers and a JSON body.
*   `client_execute_core_test.go/TestExecuteFile_MultipleRequests -> 29-36 (Multiple Requests), 38-64 (Request Structure)`

### TestExecuteFile_RequestWithError
Tests behavior when one request in a file results in an HTTP error.
*   `client_execute_core_test.go/TestExecuteFile_RequestWithError -> 29-36 (Multiple Requests - Error Handling during execution)`

### TestExecuteFile_ParseError
Tests behavior when the `.http` file itself has a syntax error.
*   `client_execute_core_test.go/TestExecuteFile_ParseError -> 7-10 (File Format - Parse Error Handling)`

### TestExecuteFile_NoRequestsInFile
Tests behavior when an `.http` file is valid but contains no actual requests.
*   `client_execute_core_test.go/TestExecuteFile_NoRequestsInFile -> 7-19 (File Format - Empty/Comment-only file)`

### TestExecuteFile_ValidThenInvalidSyntax
Tests a file with a valid request followed by syntactically incorrect content.
*   `client_execute_core_test.go/TestExecuteFile_ValidThenInvalidSyntax -> 29-36 (Multiple Requests - Partial Parse Error Handling)`

### TestExecuteFile_MultipleErrors
Tests a file where multiple requests result in execution errors.
*   `client_execute_core_test.go/TestExecuteFile_MultipleErrors -> 29-36 (Multiple Requests - Multiple Execution Error Handling)`

### TestExecuteFile_CapturesResponseHeaders
Tests that the client correctly captures response headers.
*   `client_execute_core_test.go/TestExecuteFile_CapturesResponseHeaders -> (Implicitly related to) 228-236 (Expected Response Headers - as this test verifies what would be validated)`

### TestExecuteFile_SimpleGetHTTP
Tests a simple GET request from an `.http` file.
*   `client_execute_core_test.go/TestExecuteFile_SimpleGetHTTP -> 38-46 (Request Line)`

### TestExecuteFile_MultipleRequests_GreaterThanTwo
Tests executing a file with more than two requests.
*   `client_execute_core_test.go/TestExecuteFile_MultipleRequests_GreaterThanTwo -> 29-36 (Multiple Requests)`


## client_execute_edgecases_test.go

### TestExecuteFile_InvalidMethodInFile
Tests behavior with an invalid HTTP method in the request line.
*   `client_execute_edgecases_test.go/TestExecuteFile_InvalidMethodInFile -> 38-46 (Request Line - Invalid Method Handling)`

### TestExecuteFile_IgnoreEmptyBlocks_Client
Table-driven test for various scenarios involving comments, empty blocks, and separators.
*   `client_execute_edgecases_test.go/TestExecuteFile_IgnoreEmptyBlocks_Client/"SCENARIO-LIB-028-004: Valid request, then separator, then only comments" -> 12-19 (Comments), 29-36 (Multiple Requests - Comments after separator)`
*   `client_execute_edgecases_test.go/TestExecuteFile_IgnoreEmptyBlocks_Client/"SCENARIO-LIB-028-005: Only comments, then separator, then valid request" -> 12-19 (Comments), 29-36 (Multiple Requests - Comments before separator)`
*   `client_execute_edgecases_test.go/TestExecuteFile_IgnoreEmptyBlocks_Client/"SCENARIO-LIB-028-006: Valid request, separator with comments, then another valid request" -> 12-19 (Comments), 29-36 (Multiple Requests - Comments with separator)`
*   `client_execute_edgecases_test.go/TestExecuteFile_IgnoreEmptyBlocks_Client/"File with only variable definitions - ExecuteFile" -> 139-146 (Variables - File Level - File with only variables, no requests)`


## client_execute_inplace_vars_test.go
This file extensively tests in-place variables (`@name = value`), their definition, usage, and interaction with other variable types.

### TestExecuteFile_InPlace_SimpleVariableInURL
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_SimpleVariableInURL -> 129-137 (Variables - In-Place), 111-117 (Variables - General Usage), 38-46 (Request Line - URL Substitution)`

### TestExecuteFile_InPlace_VariableInHeader
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableInHeader -> 129-137 (Variables - In-Place), 111-117 (Variables - General Usage), 48-56 (Request Headers - Header Substitution)`

### TestExecuteFile_InPlace_VariableInBody
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableInBody -> 129-137 (Variables - In-Place), 111-117 (Variables - General Usage), 58-64 (Request Body - Body Substitution)`

### TestExecuteFile_InPlace_VariableDefinedByAnotherVariable
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByAnotherVariable -> 129-137 (Variables - In-Place - Chaining Definitions)`

### TestExecuteFile_InPlace_VariablePrecedenceOverEnvironment
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariablePrecedenceOverEnvironment -> 129-137 (Variables - In-Place), 208-217 (Variable Precedence - In-place over Environment)`

### TestExecuteFile_InPlace_VariableInCustomHeader
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableInCustomHeader -> 129-137 (Variables - In-Place), 48-56 (Request Headers - Custom Header Substitution)`

### TestExecuteFile_InPlace_VariableSubstitutionInBody
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableSubstitutionInBody -> 129-137 (Variables - In-Place), 58-64 (Request Body - JSON Body Substitution)`

### TestExecuteFile_InPlace_VariableDefinedBySystemVariable
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedBySystemVariable -> 129-137 (Variables - In-Place - Definition with System Variable), 148-157 (Variables - System Variables), 159-163 ({{$uuid}})`

### TestExecuteFile_InPlace_VariableDefinedByOsEnvVariable
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByOsEnvVariable -> 129-137 (Variables - In-Place - Definition with System Variable), 188-195 ({{$env}})`

### TestExecuteFile_InPlace_VariableInAuthHeader
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableInAuthHeader -> 129-137 (Variables - In-Place), 48-56 (Request Headers - Authorization Header Substitution)`

### TestExecuteFile_InPlace_VariableInJsonRequestBody
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableInJsonRequestBody -> 129-137 (Variables - In-Place), 58-64 (Request Body - JSON Structure Substitution)`

### TestExecuteFile_InPlace_VariableDefinedByAnotherInPlaceVariable
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByAnotherInPlaceVariable -> 129-137 (Variables - In-Place - Chaining Definitions)`

### TestExecuteFile_InPlace_VariableDefinedByDotEnvOsVariable
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByDotEnvOsVariable -> 129-137 (Variables - In-Place - Definition with System Variable), 197-206 ({{$dotenv}})`

### TestExecuteFile_InPlace_Malformed_NameOnlyNoEqualsNoValue
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_Malformed_NameOnlyNoEqualsNoValue -> 129-137 (Variables - In-Place - Malformed Definition Error Handling)`

### TestExecuteFile_InPlace_Malformed_NoNameEqualsValue
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_Malformed_NoNameEqualsValue -> 129-137 (Variables - In-Place - Malformed Definition Error Handling)`

### TestExecuteFile_InPlace_VariableDefinedByDotEnvSystemVariable
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByDotEnvSystemVariable -> 129-137 (Variables - In-Place - Definition with System Variable), 197-206 ({{$dotenv}})`

### TestExecuteFile_InPlace_VariableDefinedByRandomInt
*   `client_execute_inplace_vars_test.go/TestExecuteFile_InPlace_VariableDefinedByRandomInt -> 129-137 (Variables - In-Place - Definition with System Variable), 182-186 ({{$randomInt}})`


## client_execute_system_vars_test.go
This file tests the direct usage of various system variables (`{{$...}}`) in requests.

### TestExecuteFile_WithGuidSystemVariable
Tests `{{$guid}}` and `{{$uuid}}` system variables in URL, header, and body. Verifies that multiple instances within the same request resolve to the same value.
*   `client_execute_system_vars_test.go/TestExecuteFile_WithGuidSystemVariable -> 159-163 ({{$uuid}} / {{$guid}})`

### TestExecuteFile_WithIsoTimestampSystemVariable
Tests `{{$isoTimestamp}}` system variable.
*   `client_execute_system_vars_test.go/TestExecuteFile_WithIsoTimestampSystemVariable -> 165-169 ({{$isoTimestamp}})`

### TestExecuteFile_WithDatetimeSystemVariables
Tests `{{$datetime "format" "offset"}}` system variable with various formats and offsets.
*   `client_execute_system_vars_test.go/TestExecuteFile_WithDatetimeSystemVariables -> 171-180 ({{$datetime}})`

### TestExecuteFile_WithTimestampSystemVariable
Tests `{{$timestamp "offset"}}` system variable (similar to `{{$datetime}}` but for Unix timestamps).
*   `client_execute_system_vars_test.go/TestExecuteFile_WithTimestampSystemVariable -> 171-180 ({{$datetime}} - Used for {{$timestamp}})`

### TestExecuteFile_WithRandomIntSystemVariable
Table-driven test for `{{$randomInt "min" "max"}}` system variable.
*   `client_execute_system_vars_test.go/TestExecuteFile_WithRandomIntSystemVariable/"valid min max args" -> 182-186 ({{$randomInt}} - Valid arguments)`
*   `client_execute_system_vars_test.go/TestExecuteFile_WithRandomIntSystemVariable/"no args" -> 182-186 ({{$randomInt}} - No arguments)`
*   `client_execute_system_vars_test.go/TestExecuteFile_WithRandomIntSystemVariable/"swapped min max args" -> 182-186 ({{$randomInt}} - Swapped/invalid arguments, literal placeholder)`
*   `client_execute_system_vars_test.go/TestExecuteFile_WithRandomIntSystemVariable/"malformed args" -> 182-186 ({{$randomInt}} - Malformed arguments, literal placeholder)`


## client_execute_vars_test.go
This file tests various variable types, their sources (system, programmatic, environment files), precedence, and usage.

### TestSubstituteDynamicSystemVariables_EnvVars
Tests direct substitution of OS environment variables using `{{$env.VAR_NAME}}`.
*   `client_execute_vars_test.go/TestSubstituteDynamicSystemVariables_EnvVars -> 188-195 ({{$env VAR_NAME}})`

### TestExecuteFile_WithCustomVariables
Tests variables provided at client initialization (e.g., `WithVars` option).
*   `client_execute_vars_test.go/TestExecuteFile_WithCustomVariables -> 119-127 (Variables - Programmatic / Client-Level), 208-217 (Variable Precedence - Programmatic variables)`

### TestExecuteFile_WithProcessEnvSystemVariable
Tests `{{$processEnv.VAR_NAME}}` for accessing OS environment variables.
*   `client_execute_vars_test.go/TestExecuteFile_WithProcessEnvSystemVariable -> 188-195 ({{$env VAR_NAME}} - $processEnv as specific syntax for OS env vars)`

### TestExecuteFile_WithDotEnvSystemVariable
Tests `{{$dotenv.VAR_NAME}}` for accessing variables from `.env` files.
*   `client_execute_vars_test.go/TestExecuteFile_WithDotEnvSystemVariable -> 197-206 ({{$dotenv VAR_NAME}})`

### TestExecuteFile_WithProgrammaticVariables
Tests variables set on the client instance after creation using `client.SetProgrammaticVar()`.
*   `client_execute_vars_test.go/TestExecuteFile_WithProgrammaticVariables -> 119-127 (Variables - Programmatic / Client-Level), 208-217 (Variable Precedence - Programmatic variables)`

### TestExecuteFile_WithLocalDatetimeSystemVariable
Tests `{{$localDatetime}}` system variable.
*   `client_execute_vars_test.go/TestExecuteFile_WithLocalDatetimeSystemVariable -> 171-180 ({{$datetime ...}} - $localDatetime as a specific form)`

### TestExecuteFile_VariableFunctionConsistency
Tests that variables resolve consistently to the same value across different parts of a single request (URL, headers, body).
*   `client_execute_vars_test.go/TestExecuteFile_VariableFunctionConsistency -> 111-117 (Variables - General Usage - Consistency of Resolution)`

### TestExecuteFile_WithHttpClientEnvJson
Tests loading variables from `http-client.env.json`, `http-client.private.env.json`, and named environment files (`http-client.env.NAME.json`) using the `WithEnvironment` client option. Covers variable precedence.
*   `client_execute_vars_test.go/TestExecuteFile_WithHttpClientEnvJson -> 139-146 (Variables - File Level - Environment Files http-client.env.json / http-client.private.env.json / http-client.env.NAME.json), 208-217 (Variable Precedence - Environment file variables)`

### TestExecuteFile_WithExtendedRandomSystemVariables
Tests extended `{{$random.*}}` system variables.
*   `client_execute_vars_test.go/TestExecuteFile_WithExtendedRandomSystemVariables/"{{$random.integer}}" -> 182-186 ({{$randomInt}} - $random.integer as alias/extension)`
*   `client_execute_vars_test.go/TestExecuteFile_WithExtendedRandomSystemVariables/"{{$random.float}}" -> 148-157 (Variables - System Variables - Extended Random: {{$random.float}} - Not explicitly documented)`
*   `client_execute_vars_test.go/TestExecuteFile_WithExtendedRandomSystemVariables/"{{$random.alphabetic}}" -> 148-157 (Variables - System Variables - Extended Random: {{$random.alphabetic}} - Not explicitly documented)`
*   `client_execute_vars_test.go/TestExecuteFile_WithExtendedRandomSystemVariables/"{{$random.alphanumeric}}" -> 148-157 (Variables - System Variables - Extended Random: {{$random.alphanumeric}} - Not explicitly documented)`
*   `client_execute_vars_test.go/TestExecuteFile_WithExtendedRandomSystemVariables/"{{$random.hexadecimal}}" -> 148-157 (Variables - System Variables - Extended Random: {{$random.hexadecimal}} - Not explicitly documented)`
*   `client_execute_vars_test.go/TestExecuteFile_WithExtendedRandomSystemVariables/"{{$random.email}}" -> 148-157 (Variables - System Variables - Extended Random: {{$random.email}} - Not explicitly documented)`


## client_init_test.go
This file tests the `NewClient` constructor and its options.

### TestNewClient
Tests the default state of a new client.
*   `client_init_test.go/TestNewClient -> Client Initialization - Default State (Not a direct syntax element but foundational for client behavior)`

### TestNewClient_WithOptions
Tests client initialization with various options:
*   `client_init_test.go/TestNewClient_WithOptions (WithBaseURL) -> 38-46 (Request Line - URL Resolution with Client BaseURL)`
*   `client_init_test.go/TestNewClient_WithOptions (WithDefaultHeader) -> 48-56 (Request Headers - Client Default Headers)`
*   `client_init_test.go/TestNewClient_WithOptions (WithHTTPClient) -> Client Initialization - Custom HTTP Client (Affects overall request execution, not a direct file syntax element but can influence behavior related to settings like timeout, redirects, cookies if custom client differs from defaults)`


## client_test_helpers_test.go
This file tests helper functions used within the test suite.
*   `TestCreateTestFileFromTemplate_DebugOutput`: Tests the `createTestFileFromTemplate` helper, specifically its debug logging. Not mapped as it tests test infrastructure rather than documented syntax.


## client_cookies_redirects_test.go
This file tests client behavior regarding HTTP cookies and redirects, including specific directives to control these behaviors.

### TestCookieJarHandling
Tests the client's cookie management.
*   `client_cookies_redirects_test.go/TestCookieJarHandling (Default Behavior) -> Client Behavior - Cookie Jar (Default handling of cookies, not a direct file syntax but foundational)`
*   `client_cookies_redirects_test.go/TestCookieJarHandling (With @no-cookie-jar) -> 240-245 (Request Settings - @no-cookie-jar)`

### TestRedirectHandling
Tests the client's redirect management.
*   `client_cookies_redirects_test.go/TestRedirectHandling (Default Behavior) -> Client Behavior - Redirect Following (Default handling of redirects, not a direct file syntax but foundational)`
*   `client_cookies_redirects_test.go/TestRedirectHandling (With @no-redirect) -> 233-238 (Request Settings - @no-redirect)`


## client_execute_config_test.go
This file appears to be intended for testing client execution with specific configurations like `WithBaseURL` and `WithDefaultHeaders`. However, the relevant test functions are currently commented out.
*   `TestExecuteFile_WithBaseURL (Commented Out)`: Would test client's `WithBaseURL` option. (Covered by `client_init_test.go/TestNewClient_WithOptions (WithBaseURL) -> 38-46`)
*   `TestExecuteFile_WithDefaultHeaders (Commented Out)`: Would test client's `WithDefaultHeader` option. (Covered by `client_init_test.go/TestNewClient_WithOptions (WithDefaultHeader) -> 48-56`)
*   `TestMinimalInConfig`: A minimal placeholder test, not mapped to specific syntax.


## client_execute_core_test.go
This file tests core client execution logic, including single/multiple requests, error handling, and basic request parsing.

### TestExecuteFile_SingleRequest
Tests execution of a file with a single, basic GET request.
*   `client_execute_core_test.go/TestExecuteFile_SingleRequest -> 29-36 (File Structure - Basic Request Definition)`

### TestExecuteFile_MultipleRequests
Tests execution of a file with two requests (GET then POST) and validates responses against an `.hresp` file.
*   `client_execute_core_test.go/TestExecuteFile_MultipleRequests -> 80-85 (Multiple Requests), 247-256 (Response Validation - .hresp files)`

### TestExecuteFile_RequestWithError
Tests execution of a file where one request fails (network error) and a subsequent one succeeds.
*   `client_execute_core_test.go/TestExecuteFile_RequestWithError -> 80-85 (Multiple Requests - Error Handling within a sequence)`

### TestExecuteFile_ParseError
Tests client behavior when an `.http` file contains syntax errors resulting in no parsable requests.
*   `client_execute_core_test.go/TestExecuteFile_ParseError -> 29-36 (File Structure - Basic Request Definition - Implied: handling of malformed files)`

### TestExecuteFile_NoRequestsInFile
Tests client behavior with a syntactically valid file that contains no executable request blocks.
*   `client_execute_core_test.go/TestExecuteFile_NoRequestsInFile -> 73-78 (Comments and Empty Lines - Files with only comments/empty lines)`

### TestExecuteFile_ValidThenInvalidSyntax
Tests execution of a file with a valid request followed by a request definition with invalid syntax.
*   `client_execute_core_test.go/TestExecuteFile_ValidThenInvalidSyntax -> 80-85 (Multiple Requests - Error Handling with mixed validity)`

### TestExecuteFile_MultipleErrors
Tests execution of a file where multiple requests encounter network errors.
*   `client_execute_core_test.go/TestExecuteFile_MultipleErrors -> 80-85 (Multiple Requests - Handling multiple errors)`

### TestExecuteFile_CapturesResponseHeaders
Tests that the client correctly captures HTTP response headers.
*   `client_execute_core_test.go/TestExecuteFile_CapturesResponseHeaders -> Client Behavior - Response Object (Capturing response headers, foundational for validation)`

### TestExecuteFile_SimpleGetHTTP
Tests a basic GET request with a full URL.
*   `client_execute_core_test.go/TestExecuteFile_SimpleGetHTTP -> 38-46 (Request Line - Method and URL)`

### TestExecuteFile_MultipleRequests_GreaterThanTwo
Tests execution of a file with three requests and validates them against an `.hresp` file.
*   `client_execute_core_test.go/TestExecuteFile_MultipleRequests_GreaterThanTwo -> 80-85 (Multiple Requests), 247-256 (Response Validation - .hresp files)`


## hresp_vars_test.go
This file tests variable definition within `.hresp` files and the core variable substitution logic used across the client.

### TestExtractHrespDefines
Tests parsing of `@name = value` definitions from `.hresp` files, likely for use within the validation logic of the `.hresp` file itself.
*   `hresp_vars_test.go/TestExtractHrespDefines -> 266-270 (Response Handler Script - Variable Definition @name = value - Syntax used in .hresp for defining local validation variables/values)`

### TestResolveAndSubstitute
This function comprehensively tests the `resolveAndSubstitute` logic, which is the engine for variable substitution in `.http` request files and potentially within `.hresp` files during validation. The sub-tests cover various variable types, sources, and behaviors:
*   `hresp_vars_test.go/TestResolveAndSubstitute (File-level variables) -> 129-137 (Variables - File Level - .http files)`
*   `hresp_vars_test.go/TestResolveAndSubstitute (Programmatic/Client-level variables) -> 119-127 (Variables - Programmatic / Client-Level)`
*   `hresp_vars_test.go/TestResolveAndSubstitute (System Variable: {{$uuid}}) -> 159-163 ({{$uuid}})`
*   `hresp_vars_test.go/TestResolveAndSubstitute (System Variable: {{$timestamp}}) -> 165-169 ({{$timestamp}})`
*   `hresp_vars_test.go/TestResolveAndSubstitute (System Variable: {{$env VAR_NAME}}) -> 188-195 ({{$env VAR_NAME}})`
*   `hresp_vars_test.go/TestResolveAndSubstitute (System Variable: {{$dotenv VAR_NAME}}) -> 197-206 ({{$dotenv VAR_NAME}})`
*   `hresp_vars_test.go/TestResolveAndSubstitute (Variable Precedence) -> 208-217 (Variable Precedence)`
*   `hresp_vars_test.go/TestResolveAndSubstitute (Fallback Values) -> 219-225 (Variables - Fallback Values)`
*   `hresp_vars_test.go/TestResolveAndSubstitute (Unresolved Variables) -> 111-117 (Variables - General Usage - Unresolved variables remain literal)`


## parser_authentication_test.go
This file tests the parsing of various authentication schemes from `.http` files.

### TestBasicAuthHeader
Tests parsing of the `Authorization: Basic <credentials>` header.
*   `parser_authentication_test.go/TestBasicAuthHeader -> 227-233 (Authentication - Basic Authentication - Via Authorization Header)`

### TestBasicAuthURL
Tests parsing of basic authentication credentials embedded in the request URL (e.g., `user:pass@domain`).
*   `parser_authentication_test.go/TestBasicAuthURL -> 227-233 (Authentication - Basic Authentication - Via URL)`

### TestBearerTokenAuth
Tests parsing of the `Authorization: Bearer <token>` header.
*   `parser_authentication_test.go/TestBearerTokenAuth -> 235-239 (Authentication - Bearer Token)`

### TestOAuthFlowWithRequestReferences
Tests an OAuth flow where a token is obtained in one request and referenced in a subsequent request's `Authorization` header.
*   `parser_authentication_test.go/TestOAuthFlowWithRequestReferences -> 241-245 (Authentication - OAuth 2.0 Flow (Implicit/Client Credentials)), 87-93 (Named Requests and Request References - Referencing previous request's response)`


## parser_request_body_test.go
This file tests the parsing of various types of request bodies.

### TestJsonRequestBodies
Tests parsing of inline JSON request bodies, including simple JSON, arrays, and nested objects.
*   `parser_request_body_test.go/TestJsonRequestBodies -> 54-60 (Request Body - Inline JSON)`

### TestFileBasedBodies
Tests using `< filepath` to specify the entire request body from an external file (e.g., JSON, text, XML).
*   `parser_request_body_test.go/TestFileBasedBodies -> 62-71 (Request Body - File Input for entire body)`

### TestFormUrlEncodedBodies
Tests parsing of inline `application/x-www-form-urlencoded` request bodies.
*   `parser_request_body_test.go/TestFormUrlEncodedBodies -> 54-60 (Request Body - Inline Form URL Encoded Data)`

### TestMultipartFormDataBodies
Tests parsing of `multipart/form-data` request bodies with inline text fields.
*   `parser_request_body_test.go/TestMultipartFormDataBodies -> 54-60 (Request Body - Inline Multipart Form Data (text fields))`

### TestFileUploadBodies
Tests parsing of `multipart/form-data` request bodies that include file uploads, where file content is referenced using `< filepath` for individual parts.
*   `parser_request_body_test.go/TestFileUploadBodies -> 54-60 (Request Body - Inline Multipart Form Data (file fields)), 62-71 (Request Body - File Input - for parts within multipart/form-data)`


## parser_request_settings_test.go
This file tests the parsing of various request-level settings directives.

### TestNameDirective
Tests parsing of the `@name <requestName>` directive to name a request.
*   `parser_request_settings_test.go/TestNameDirective -> 87-93 (Named Requests and Request References - Defining a Named Request)`

### TestNoRedirectDirective
Tests parsing of the `@no-redirect` directive to disable following HTTP redirects for a request.
*   `parser_request_settings_test.go/TestNoRedirectDirective -> 95-99 (Request Settings - @no-redirect)`

### TestNoCookieJarDirective
Tests parsing of the `@no-cookie-jar` directive to disable the use of the client's cookie jar for a request.
*   `parser_request_settings_test.go/TestNoCookieJarDirective -> 101-105 (Request Settings - @no-cookie-jar)`

### TestTimeoutDirective
Tests parsing of the `@timeout <milliseconds>` directive to set a custom timeout for a request.
*   `parser_request_settings_test.go/TestTimeoutDirective -> 107-109 (Request Settings - @timeout N)`


## parser_request_structure_test.go
This file tests basic request structure parsing, including request separation and comment handling.

### TestParserRequestNaming
Tests the `### Request Name` syntax for separating and naming requests within a single `.http` file.
*   `parser_request_structure_test.go/TestParserRequestNaming -> 80-85 (Multiple Requests - Request Separator ###)`

### TestParserCommentStyles
Tests that both `#` and `//` style comments are correctly handled and do not interfere with the parsing of requests or directives.
*   `parser_request_structure_test.go/TestParserCommentStyles -> 73-78 (Comments and Empty Lines - # and // style comments)`


## parser_response_validation_test.go
This file tests the parsing of response validation placeholders in `.hresp` files and the parsing of chained requests in `.http` files.

### TestParseResponseValidationPlaceholder_Any
Tests parsing of the `{{$any}}` placeholder in `.hresp` files for flexible value matching.
*   `parser_response_validation_test.go/TestParseResponseValidationPlaceholder_Any -> 247-256 (Response Validation - .hresp files - Placeholder: {{$any}})`

### TestParseResponseValidationPlaceholder_Regexp
Tests parsing of the `{{$regexp 'pattern'}}` placeholder in `.hresp` files for regex-based value matching.
*   `parser_response_validation_test.go/TestParseResponseValidationPlaceholder_Regexp -> 247-256 (Response Validation - .hresp files - Placeholder: {{$regexp 'pattern'}})`

### TestParseResponseValidationPlaceholder_AnyGuid
Tests parsing of the `{{$anyGuid}}` placeholder in `.hresp` files for matching UUID/GUID values.
*   `parser_response_validation_test.go/TestParseResponseValidationPlaceholder_AnyGuid -> 247-256 (Response Validation - .hresp files - Placeholder: {{$anyGuid}})`

### TestParseResponseValidationPlaceholder_AnyTimestamp
Tests parsing of the `{{$anyTimestamp}}` placeholder in `.hresp` files for matching Unix timestamp values.
*   `parser_response_validation_test.go/TestParseResponseValidationPlaceholder_AnyTimestamp -> 247-256 (Response Validation - .hresp files - Placeholder: {{$anyTimestamp}})`

### TestParseResponseValidationPlaceholder_AnyDatetime
Tests parsing of the `{{$anyDatetime 'format'}}` placeholder in `.hresp` files for matching datetime strings against a specified format.
*   `parser_response_validation_test.go/TestParseResponseValidationPlaceholder_AnyDatetime -> 247-256 (Response Validation - .hresp files - Placeholder: {{$anyDatetime 'format'}})`

### TestParseChainedRequests
Tests parsing of `.http` files where requests reference data from responses of preceding named requests (e.g., `{{requestName.response.body.field}}`).
*   `parser_response_validation_test.go/TestParseChainedRequests -> 87-93 (Named Requests and Request References - Referencing previous request's response)`


## parser_environment_vars_test.go
This file tests the parsing of environment variables and in-file variable definitions.

### TestParseRequestFile_EnvironmentVariables
Tests the parser's ability to recognize environment variable placeholders (`{{env_var}}` and `{{env_var:default_value}}`) in request URLs, query parameters, headers, and bodies. It verifies that these placeholders are preserved in the parsed request structure for later substitution.
*   `parser_environment_vars_test.go/TestParseRequestFile_EnvironmentVariables -> 123-130 (Variables - Environment Variables - Basic Usage)`
*   `parser_environment_vars_test.go/TestParseRequestFile_EnvironmentVariables -> 132-137 (Variables - Environment Variables - Default Values)`
*   `parser_environment_vars_test.go/TestParseRequestFile_EnvironmentVariables -> 158-168 (Variables - Variable Usage - In Request URL)`
*   `parser_environment_vars_test.go/TestParseRequestFile_EnvironmentVariables -> 158-168 (Variables - Variable Usage - In Query Parameters)`
*   `parser_environment_vars_test.go/TestParseRequestFile_EnvironmentVariables -> 158-168 (Variables - Variable Usage - In Headers)`
*   `parser_environment_vars_test.go/TestParseRequestFile_EnvironmentVariables -> 158-168 (Variables - Variable Usage - In Request Body)`

### TestParseRequestFile_VariableDefinitions
Tests the parsing of in-file variable definitions using the `@name = value` syntax and their subsequent use in requests within the same file.
*   `parser_environment_vars_test.go/TestParseRequestFile_VariableDefinitions -> 111-121 (Variables - File Variables (@name = value))`
*   `parser_environment_vars_test.go/TestParseRequestFile_VariableDefinitions -> 158-168 (Variables - Variable Usage - In Request URL)`
*   `parser_environment_vars_test.go/TestParseRequestFile_VariableDefinitions -> 158-168 (Variables - Variable Usage - In Headers)`


## parser_system_vars_test.go
This file tests the parsing of various system variables used in HTTP requests.

### TestParseRequestFile_BasicSystemVariables
Tests the parser's ability to recognize and preserve basic system variable placeholders like `{{$guid}}`, `{{$uuid}}`, `{{$timestamp}}`, `{{$isoTimestamp}}`, `{{$datetime "format"}}`, and `{{$localDatetime "format"}}` in request URLs and headers.
*   `parser_system_vars_test.go/TestParseRequestFile_BasicSystemVariables -> 139-156 (Variables - System Variables - Common System Variables)`
*   `parser_system_vars_test.go/TestParseRequestFile_BasicSystemVariables -> 139-156 (Variables - System Variables - Datetime Variables)`

### TestParseRequestFile_RandomSystemVariables
Tests the parser's ability to recognize and preserve random value generating system variable placeholders such as `{{$randomInt}}`, `{{$randomInt min max}}`, `{{$random.integer(min, max)}}`, `{{$random.float(min, max)}}`, `{{$random.alphabetic(length)}}`, `{{$random.alphanumeric(length)}}`, and `{{$random.hexadecimal(length)}}`.
*   `parser_system_vars_test.go/TestParseRequestFile_RandomSystemVariables -> 139-156 (Variables - System Variables - Random Value Generators)`

### TestParseRequestFile_EnvironmentAccess
Tests the parser's ability to recognize and preserve system variables used for accessing environment variables, specifically `{{$processEnv VAR_NAME}}` and `{{$env.VAR_NAME}}`.
*   `parser_system_vars_test.go/TestParseRequestFile_EnvironmentAccess -> 139-156 (Variables - System Variables - Environment Access)`


## parser_test.go
This file contains miscellaneous parser tests, including handling of empty blocks, comments around separators, parsing `.hresp` files, variable scoping, and import directives.

### TestParseRequests_IgnoreEmptyBlocks
Tests that the parser correctly ignores blocks within an `.http` file that are empty or contain only comments and/or `###` separators, ensuring no erroneous requests are generated.
*   `parser_test.go/TestParseRequests_IgnoreEmptyBlocks -> 73-78 (Comments and Empty Lines - Empty lines and blocks)`

### TestParseRequests_SeparatorComments
Tests the parser's behavior when comments (both `#` and `//` styles) are placed before, after, or on the same line as the `###` request separator in `.http` files. It ensures that requests are still correctly separated and parsed.
*   `parser_test.go/TestParseRequests_SeparatorComments -> 80-85 (Multiple Requests - Request Separator ###)`

### TestParseExpectedResponses_SeparatorComments
Tests the parser's behavior for `.hresp` files when comments are placed around the `###` response separator. It ensures that multiple expected response blocks are correctly parsed.
*   `parser_test.go/TestParseExpectedResponses_SeparatorComments -> 218-225 (Response Validation - .hresp files - Structure and Separators)`

### TestParseExpectedResponses_Simple
Tests the parsing of various components within `.hresp` files, including status code, headers (with and without variable placeholders), and body content (JSON, plain text, with variable placeholders).
*   `parser_test.go/TestParseExpectedResponses_Simple -> 218-225 (Response Validation - .hresp files - Structure and Separators)`
*   `parser_test.go/TestParseExpectedResponses_Simple -> 227-231 (Response Validation - .hresp files - Status Code Validation)`
*   `parser_test.go/TestParseExpectedResponses_Simple -> 233-239 (Response Validation - .hresp files - Header Validation)`
*   `parser_test.go/TestParseExpectedResponses_Simple -> 241-245 (Response Validation - .hresp files - Body Validation)`
*   `parser_test.go/TestParseExpectedResponses_Simple -> 158-168 (Variables - Variable Usage - In Headers)`
*   `parser_test.go/TestParseExpectedResponses_Simple -> 158-168 (Variables - Variable Usage - In Request Body)`

### TestParseRequestFile_VariableScoping
Tests the scoping rules for variables defined within `.http` files, including how file-level variables can be referenced and potentially overridden by request-specific variables, and how variables can reference other variables.
*   `parser_test.go/TestParseRequestFile_VariableScoping -> 169-176 (Variables - Variable Precedence and Scoping)`


## validator_body_test.go
This file tests the response body validation logic, primarily focusing on exact matching when using `.hresp` files.

### TestValidateResponses_Body_ExactMatch
Tests the exact matching of the actual response body against the body specified in an `.hresp` file. Covers scenarios like matching bodies, mismatching bodies, and handling of empty bodies in either the actual response or the `.hresp` file.
*   `validator_body_test.go/TestValidateResponses_Body_ExactMatch -> 241-245 (Response Validation - .hresp files - Body Validation)`


## validator_general_test.go
This file contains general tests for the response validation logic using `.hresp` files, including full and partial expectation matching.

### TestValidateResponses_WithSampleFile
Tests the response validator against a fully defined `.hresp` file (`testdata/http_response_files/sample1.http`). It covers scenarios such as a perfect match between the actual response and the expected response, as well as various mismatches including status code, status string, header values, missing headers, and body content.
*   `validator_general_test.go/TestValidateResponses_WithSampleFile -> 218-225 (Response Validation - .hresp files - Structure and Separators)`
*   `validator_general_test.go/TestValidateResponses_WithSampleFile -> 227-231 (Response Validation - .hresp files - Status Code Validation)`
*   `validator_general_test.go/TestValidateResponses_WithSampleFile -> 233-239 (Response Validation - .hresp files - Header Validation)`
*   `validator_general_test.go/TestValidateResponses_WithSampleFile -> 241-245 (Response Validation - .hresp files - Body Validation)`

### TestValidateResponses_PartialExpected
Tests the response validator's behavior when the `.hresp` file defines only a partial set of expectations. For example, an `.hresp` file might only specify an expected status code, or only certain headers, implying that other aspects (like an empty body if not specified alongside status/headers) are also part of the expectation. This test ensures that only the explicitly defined elements (and their sensible defaults/implications) are validated.
*   `validator_general_test.go/TestValidateResponses_PartialExpected -> 227-231 (Response Validation - .hresp files - Status Code Validation)`
*   `validator_general_test.go/TestValidateResponses_PartialExpected -> 233-239 (Response Validation - .hresp files - Header Validation)`
*   `validator_general_test.go/TestValidateResponses_PartialExpected -> 241-245 (Response Validation - .hresp files - Body Validation)`


## validator_headers_test.go
This file tests the validation of HTTP response headers against expectations defined in `.hresp` files.

### TestValidateResponses_Headers
This function tests various aspects of header validation when comparing an actual HTTP response to an `.hresp` file. Scenarios include:
- Perfect match of all expected headers and their values.
- Mismatched value for an expected header.
- An expected header (defined in `.hresp`) is missing from the actual response.
- Actual response contains extra headers not defined in `.hresp` (these should be ignored).
- Correct validation of multi-value headers, including cases where the order might differ but all expected values are present, or where the actual response contains a superset of the expected multi-values.
- Case-insensitive matching for header keys.
*   `validator_headers_test.go/TestValidateResponses_Headers -> 233-239 (Response Validation - .hresp files - Header Validation)`


## validator_status_test.go
This file tests the validation of HTTP response status codes and status strings (reason phrases) against expectations defined in `.hresp` files.

### TestValidateResponses_StatusString
Tests the validation of the full HTTP status string (e.g., "200 OK", "404 Not Found") from an actual response against the status line specified in an `.hresp` file. It covers scenarios where the `.hresp` file might specify the full status string or only the status code, and how the validator matches the actual response's status string accordingly.
*   `validator_status_test.go/TestValidateResponses_StatusString -> 227-231 (Response Validation - .hresp files - Status Code Validation)`

### TestValidateResponses_StatusCode
Tests the validation of the numeric HTTP status code (e.g., 200, 404) from an actual response against the status code specified in an `.hresp` file. This includes cases of matching and mismatching codes, and how the validation behaves if the `.hresp` file only contains a status code without a reason phrase.
*   `validator_status_test.go/TestValidateResponses_StatusCode -> 227-231 (Response Validation - .hresp files - Status Code Validation)`


## validator_placeholders_test.go
This file tests the validation of response bodies using various dynamic placeholders (e.g., `$any`, `$regexp()`, `$any(guid)`) within `.hresp` files. These placeholders allow for flexible matching of content that may not be known beforehand or requires pattern validation.

### TestValidateResponses_BodyRegexpPlaceholder
Validates the `$regexp(pattern)` placeholder, ensuring it correctly matches parts of the response body against the provided regular expression. It covers successful matches, mismatches, and error handling for invalid regex patterns in the `.hresp` file.
*   `validator_placeholders_test.go/TestValidateResponses_BodyRegexpPlaceholder -> 259-261 (Response Validation - .hresp files - Body Validation - Placeholders - $regexp(pattern))`

### TestValidateResponses_BodyAnyGuidPlaceholder
Validates the `$any(guid)` (or `$any(uuid)`) placeholder, checking if it correctly identifies and matches valid UUIDs/GUIDs within the response body.
*   `validator_placeholders_test.go/TestValidateResponses_BodyAnyGuidPlaceholder -> 249-250 (Response Validation - .hresp files - Body Validation - Placeholders - $any(uuid) / $any(guid))`

### TestValidateResponses_BodyAnyTimestampPlaceholder
Validates the `$any(timestamp)` placeholder, ensuring it correctly matches Unix timestamps (integer seconds since epoch) in the response body.
*   `validator_placeholders_test.go/TestValidateResponses_BodyAnyTimestampPlaceholder -> 257-258 (Response Validation - .hresp files - Body Validation - Placeholders - $any(timestamp))`

### TestValidateResponses_BodyAnyDatetimePlaceholder
Validates the `$any(datetime, format?)` placeholder, testing its ability to match date-time strings. This includes default RFC3339 format and custom-defined formats, as well as error handling for non-matching values or invalid format strings.
*   `validator_placeholders_test.go/TestValidateResponses_BodyAnyDatetimePlaceholder -> 254-256 (Response Validation - .hresp files - Body Validation - Placeholders - $any(datetime, format?))`

### TestValidateResponses_BodyAnyPlaceholder
Validates the basic `$any` placeholder, which is designed to match any sequence of characters (non-greedily). Tests cover matching simple strings, strings with special characters, empty segments, multi-line content, and scenarios with multiple `$any` placeholders.
*   `validator_placeholders_test.go/TestValidateResponses_BodyAnyPlaceholder -> 242-243 (Response Validation - .hresp files - Body Validation - Placeholders - $any)`


## validator_setup_test.go
This file tests edge cases and error handling scenarios related to the setup and inputs for the response validation process, specifically concerning the actual responses provided to the validator and the integrity of the `.hresp` file. These tests ensure the robustness of the validation mechanism itself.

### TestValidateResponses_NilAndEmptyActuals
Tests how the `ValidateResponses` function handles scenarios where the provided actual `Response` objects are problematic. This includes passing a `nil` slice of responses, an empty slice, or a slice containing `nil` `Response` objects. The primary focus is on ensuring correct error reporting when the count of actual responses does not match the count of expected responses derived from the `.hresp` file.
*   This test validates the operational robustness of the response validation mechanism regarding input arguments and does not directly map to a specific HTTP syntax feature in `docs/http_syntax.md`.

### TestValidateResponses_FileErrors
Tests how the `ValidateResponses` function handles errors related to the `.hresp` file itself. This includes scenarios where the specified `.hresp` file is missing, is empty (contains no expected responses), or is malformed (e.g., contains an invalid status line). The test ensures appropriate errors are reported for such file-related issues.
*   This test validates the error handling of the response validation mechanism concerning the integrity and accessibility of the `.hresp` file and does not directly map to a specific HTTP syntax feature in `docs/http_syntax.md`.


























