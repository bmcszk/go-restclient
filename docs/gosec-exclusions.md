# gosec exclusions (CI: `-exclude G404,G304`)

## G404 — weak random number source (math/rand)

All uses generate **faker/test data**: `{{$randomWord}}`, `{{$randomInt}}`, `{{$randomColor}}`,
MAC/domain/URL generators, boundary-value picks for validator fuzz data. None of it is security
material — tokens and secrets never come from these generators. `math/rand` is the correct
(non-crypto, fast) choice for data generation.

## G304 — file inclusion via variable

Reading files from user-supplied paths is the tool's **core purpose**: `restclient file.http`,
`--env staging`, `@ref` fixtures. A local CLI opening the paths its user names is intended
behavior, not path traversal — there is no remote attack surface.
