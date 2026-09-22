package restclient

// Random-value substitution: faker variables, $random.* variables, and charset-based generators.

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Name lists for person data generation
var firstNames = []string{
	"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda", "William", "Elizabeth",
	"David", "Barbara", "Richard", "Susan", "Joseph", "Jessica", "Thomas", "Sarah", "Christopher", "Karen",
	"Charles", "Helen", "Daniel", "Nancy", "Matthew", "Betty", "Anthony", "Dorothy", "Mark", "Lisa",
	"Donald", "Sandra", "Steven", "Donna", "Paul", "Carol", "Andrew", "Ruth", "Joshua", "Sharon",
}

var lastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez",
	"Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson", "Martin",
	"Lee", "Perez", "Thompson", "White", "Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson",
	"Walker", "Young", "Allen", "King", "Wright", "Scott", "Torres", "Nguyen", "Hill", "Flores",
}

// Job titles for business data
var jobTitles = []string{
	"Software Engineer", "Product Manager", "Data Scientist", "UX Designer", "DevOps Engineer",
	"Marketing Manager", "Sales Representative", "Project Manager", "Business Analyst", "QA Engineer",
	"Frontend Developer", "Backend Developer", "Full Stack Developer", "Technical Writer", "Architect",
	"Consultant", "Analyst", "Director", "Manager", "Coordinator", "Specialist", "Administrator",
	"Executive", "Lead", "Senior Developer", "Junior Developer", "Intern", "VP of Engineering",
}

// Contact data lists
var streetNames = []string{
	"Main St", "Oak Ave", "Pine St", "Maple Ave", "Cedar St", "Elm St", "Washington Ave", "Park Ave",
	"First St", "Second St", "Third St", "Market St", "Church St", "Broad St", "High St", "King St",
	"Mill St", "Water St", "School St", "State St", "North St", "South St", "East St", "West St",
	"Center St", "Union St", "Bridge St", "Franklin St", "Lincoln Ave", "Madison Ave", "Adams St",
}

var cities = []string{
	"New York", "Los Angeles", "Chicago", "Houston", "Phoenix", "Philadelphia", "San Antonio", "San Diego",
	"Dallas", "San Jose", "Austin", "Jacksonville", "Fort Worth", "Columbus", "Charlotte", "San Francisco",
	"Indianapolis", "Seattle", "Denver", "Washington", "Boston", "El Paso", "Nashville", "Detroit",
	"Oklahoma City", "Portland", "Las Vegas", "Memphis", "Louisville", "Baltimore", "Milwaukee", "Albuquerque",
	"Tucson", "Fresno", "Sacramento", "Mesa", "Kansas City", "Atlanta", "Long Beach", "Colorado Springs",
}

var states = []string{
	"Alabama", "Alaska", "Arizona", "Arkansas", "California", "Colorado", "Connecticut", "Delaware",
	"Florida", "Georgia", "Hawaii", "Idaho", "Illinois", "Indiana", "Iowa", "Kansas", "Kentucky",
	"Louisiana", "Maine", "Maryland", "Massachusetts", "Michigan", "Minnesota", "Mississippi", "Missouri",
	"Montana", "Nebraska", "Nevada", "New Hampshire", "New Jersey", "New Mexico", "New York",
	"North Carolina", "North Dakota", "Ohio", "Oklahoma", "Oregon", "Pennsylvania", "Rhode Island",
	"South Carolina", "South Dakota", "Tennessee", "Texas", "Utah", "Vermont", "Virginia", "Washington",
	"West Virginia", "Wisconsin", "Wyoming",
}

var countries = []string{
	"United States", "Canada", "United Kingdom", "Germany", "France", "Italy", "Spain", "Netherlands",
	"Belgium", "Switzerland", "Austria", "Sweden", "Norway", "Denmark", "Finland", "Portugal", "Ireland",
	"Australia", "New Zealand", "Japan", "South Korea", "Singapore", "Brazil", "Mexico", "Argentina",
}

// Internet data lists
var domains = []string{
	"example.com", "test.org", "demo.net", "sample.co", "fake.io", "mock.dev", "placeholder.site",
	"tempmail.com", "fakesite.org", "testdomain.net", "randomsite.com", "demopage.org",
}

var protocols = []string{"http", "https"}

var paths = []string{
	"/api/v1", "/dashboard", "/profile", "/settings", "/docs", "/help", "/contact", "/about",
	"/products", "/services", "/blog", "/news", "/search", "/login", "/register", "/admin",
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:89.0) Gecko/20100101 Firefox/89.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 " +
		"(KHTML, like Gecko) Version/14.1.1 Safari/605.1.15",
}

// fakerReplacement pairs one faker variable regex with its value generator.
type fakerReplacement struct {
	re *regexp.Regexp
	fn func(string) string
}

// fakerReplacements lists all faker tokens in substitution order: VS Code style first, then JetBrains style.
var fakerReplacements = []fakerReplacement{
	// Person data - VS Code style
	{reRandomFirstName, getRandomFirstName},
	{reRandomLastName, getRandomLastName},
	{reRandomFullName, getRandomFullName},
	{reRandomJobTitle, getRandomJobTitle},
	// Contact data - VS Code style
	{reRandomPhoneNumber, getRandomPhoneNumber},
	{reRandomStreetAddress, getRandomStreetAddress},
	{reRandomCity, getRandomCity},
	{reRandomState, getRandomState},
	{reRandomZipCode, getRandomZipCode},
	{reRandomCountry, getRandomCountry},
	// Internet data - VS Code style
	{reRandomUrl, getRandomUrl},
	{reRandomDomainName, getRandomDomainName},
	{reRandomUserAgent, getRandomUserAgent},
	{reRandomMacAddress, getRandomMacAddress},
	// Person data - JetBrains style
	{reRandomFirstNameDot, getRandomFirstName},
	{reRandomLastNameDot, getRandomLastName},
	{reRandomFullNameDot, getRandomFullName},
	{reRandomJobTitleDot, getRandomJobTitle},
	// Contact data - JetBrains style
	{reRandomPhoneNumberDot, getRandomPhoneNumber},
	{reRandomStreetAddressDot, getRandomStreetAddress},
	{reRandomCityDot, getRandomCity},
	{reRandomStateDot, getRandomState},
	{reRandomZipCodeDot, getRandomZipCode},
	{reRandomCountryDot, getRandomCountry},
	// Internet data - JetBrains style
	{reRandomUrlDot, getRandomUrl},
	{reRandomDomainNameDot, getRandomDomainName},
	{reRandomUserAgentDot, getRandomUserAgent},
	{reRandomMacAddressDot, getRandomMacAddress},
}

// substituteFakerVariables handles the substitution of faker/person data variables.
func substituteFakerVariables(text string) string {
	for _, r := range fakerReplacements {
		text = r.re.ReplaceAllStringFunc(text, r.fn)
	}
	return text
}

// getRandomFirstName returns a random first name
func getRandomFirstName(_ string) string {
	if len(firstNames) > 0 {
		return firstNames[rand.Intn(len(firstNames))]
	}
	return "John"
}

// getRandomLastName returns a random last name
func getRandomLastName(_ string) string {
	if len(lastNames) > 0 {
		return lastNames[rand.Intn(len(lastNames))]
	}
	return "Doe"
}

// getRandomFullName returns a random full name
func getRandomFullName(_ string) string {
	firstName := "John"
	lastName := "Doe"
	if len(firstNames) > 0 {
		firstName = firstNames[rand.Intn(len(firstNames))]
	}
	if len(lastNames) > 0 {
		lastName = lastNames[rand.Intn(len(lastNames))]
	}
	return firstName + " " + lastName
}

// getRandomJobTitle returns a random job title
func getRandomJobTitle(_ string) string {
	if len(jobTitles) > 0 {
		return jobTitles[rand.Intn(len(jobTitles))]
	}
	return "Software Engineer"
}

// Contact data generators

// getRandomPhoneNumber returns a random phone number
func getRandomPhoneNumber(_ string) string {
	areaCode := rand.Intn(900) + 100 // 100-999
	exchange := rand.Intn(900) + 100 // 100-999
	number := rand.Intn(10000)       // 0000-9999
	return fmt.Sprintf("(%03d) %03d-%04d", areaCode, exchange, number)
}

// getRandomStreetAddress returns a random street address
func getRandomStreetAddress(_ string) string {
	if len(streetNames) == 0 {
		return "123 Main St"
	}
	streetNumber := rand.Intn(9999) + 1 // 1-9999
	streetName := streetNames[rand.Intn(len(streetNames))]
	return fmt.Sprintf("%d %s", streetNumber, streetName)
}

// getRandomCity returns a random city
func getRandomCity(_ string) string {
	if len(cities) > 0 {
		return cities[rand.Intn(len(cities))]
	}
	return "New York"
}

// getRandomState returns a random state
func getRandomState(_ string) string {
	if len(states) > 0 {
		return states[rand.Intn(len(states))]
	}
	return "California"
}

// getRandomZipCode returns a random ZIP code
func getRandomZipCode(_ string) string {
	zipCode := rand.Intn(100000) // 00000-99999
	return fmt.Sprintf("%05d", zipCode)
}

// getRandomCountry returns a random country
func getRandomCountry(_ string) string {
	if len(countries) > 0 {
		return countries[rand.Intn(len(countries))]
	}
	return "United States"
}

// Internet data generators

// getRandomUrl returns a random URL
func getRandomUrl(_ string) string {
	if len(protocols) == 0 || len(domains) == 0 || len(paths) == 0 {
		return "https://example.com/api"
	}
	protocol := protocols[rand.Intn(len(protocols))]
	domain := domains[rand.Intn(len(domains))]
	path := paths[rand.Intn(len(paths))]
	return fmt.Sprintf("%s://%s%s", protocol, domain, path)
}

// getRandomDomainName returns a random domain name
func getRandomDomainName(_ string) string {
	if len(domains) > 0 {
		return domains[rand.Intn(len(domains))]
	}
	return "example.com"
}

// getRandomUserAgent returns a random user agent string
func getRandomUserAgent(_ string) string {
	if len(userAgents) > 0 {
		return userAgents[rand.Intn(len(userAgents))]
	}
	return "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
}

// getRandomMacAddress returns a random MAC address
func getRandomMacAddress(_ string) string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		rand.Intn(256), rand.Intn(256), rand.Intn(256),
		rand.Intn(256), rand.Intn(256), rand.Intn(256))
}

// randomStringFromCharset generates a random string of a given length using characters from the provided charset.
func randomStringFromCharset(length int, charset string) string {
	if length <= 0 || len(charset) == 0 { // Added len(charset) == 0 check
		return ""
	}
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}
	return string(b)
}

// substituteRandomVariables handles the substitution of $random.* variables.
func substituteRandomVariables(text string, programmaticVars map[string]any) string {
	// Integer types
	text = reRandomInt.ReplaceAllStringFunc(text,
		substituteRandomIntFunc(reRandomInt, defaultRandomMinInt, defaultRandomMaxInt))
	text = reRandomDotInteger.ReplaceAllStringFunc(text,
		substituteRandomIntFunc(reRandomDotInteger, defaultRandomMinInt, defaultRandomMaxInt))

	// Float types
	text = reRandomFloat.ReplaceAllStringFunc(text,
		substituteRandomFloatFunc(reRandomFloat, defaultRandomMinFloat, defaultRandomMaxFloat))
	text = reRandomDotFloat.ReplaceAllStringFunc(text,
		substituteRandomFloatFunc(reRandomDotFloat, defaultRandomMinFloat, defaultRandomMaxFloat))

	// Boolean
	text = strings.ReplaceAll(text, "{{$randomBoolean}}", strconv.FormatBool(rand.Intn(2) == 0))

	// Hexadecimal
	text = reRandomHex.ReplaceAllStringFunc(text, substituteRandomHexHelper(reRandomHex, defaultRandomHexLength))
	text = reRandomDotHexadecimal.ReplaceAllStringFunc(text,
		substituteRandomHexHelper(reRandomDotHexadecimal, defaultRandomHexLength))

	// Alphabetic / Alphanumeric
	text = reRandomDotAlphabetic.ReplaceAllStringFunc(text,
		substituteRandomLengthCharsetFunc(reRandomDotAlphabetic, charsetAlphabetic))
	// Uses underscore
	text = reRandomAlphaNumeric.ReplaceAllStringFunc(text,
		substituteRandomLengthCharsetFunc(reRandomAlphaNumeric, charsetAlphaNumericWithExtra))
	// No underscore
	text = reRandomDotAlphanumeric.ReplaceAllStringFunc(text,
		substituteRandomLengthCharsetFunc(reRandomDotAlphanumeric, charsetAlphaNumeric))

	// General Random String
	text = reRandomString.ReplaceAllStringFunc(text, substituteRandomLengthCharsetFunc(reRandomString, charsetFull))

	// Email
	emailGenerator := func() string {
		return fmt.Sprintf("%s@%s.com",
			randomStringFromCharset(10, charsetAlphaNumeric),
			randomStringFromCharset(7, charsetAlphabetic))
	}
	text = strings.ReplaceAll(text, "{{$randomEmail}}", emailGenerator())
	text = strings.ReplaceAll(text, "{{$random.email}}", emailGenerator())

	// Domain
	text = strings.ReplaceAll(text, "{{$randomDomain}}",
		fmt.Sprintf("%s.com", randomStringFromCharset(10, charsetAlphabetic)))

	// IP Addresses
	text = strings.ReplaceAll(text, "{{$randomIPv4}}",
		fmt.Sprintf("%d.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256)))

	text = strings.ReplaceAll(text, "{{$randomIPv6}}", func() string {
		segments := make([]string, 8)
		for i := 0; i < 8; i++ {
			segments[i] = fmt.Sprintf("%x", rand.Intn(0x10000))
		}
		return strings.Join(segments, ":")
	}())

	// UUID
	text = strings.ReplaceAll(text, "{{$randomUUID}}", uuid.New().String())

	// Password (uses programmaticVars, so it calls the existing substituteRandomPasswordFunc with modification)
	text = reRandomPassword.ReplaceAllStringFunc(text, func(match string) string {
		return substituteRandomPasswordFunc(match, programmaticVars)
	})

	// Color
	text = strings.ReplaceAll(text, "{{$randomColor}}",
		fmt.Sprintf("#%02x%02x%02x", rand.Intn(256), rand.Intn(256), rand.Intn(256)))

	// Word
	if len(randomWords) > 0 { // Prevent panic on empty slice
		text = strings.ReplaceAll(text, "{{$randomWord}}", randomWords[rand.Intn(len(randomWords))])
	}

	// Person/Identity data (faker variables)
	text = substituteFakerVariables(text)

	return text
}

// substituteRandomPasswordFunc handles $randomPassword.* substitution; programmaticVars enables charset overrides.
func substituteRandomPasswordFunc(match string, programmaticVars map[string]any) string {
	length := parsePasswordLength(match)
	if length < 0 {
		return match // Malformed length
	}
	if length == 0 {
		return ""
	}

	charset := getPasswordCharset(programmaticVars)
	return randomStringFromCharset(length, charset)
}

// parsePasswordLength extracts and validates the length parameter from a password match
func parsePasswordLength(match string) int {
	parts := reRandomPassword.FindStringSubmatch(match)
	length := defaultRandomPasswordLength
	if len(parts) >= 2 && parts[1] != "" {
		parsedLen, err := strconv.Atoi(parts[1])
		if err != nil || parsedLen < 0 {
			return -1 // Invalid length
		}
		length = parsedLen
	}
	return length
}

// getPasswordCharset determines the charset to use for password generation
func getPasswordCharset(programmaticVars map[string]any) string {
	if psVars, ok := programmaticVars["password"]; ok {
		if psMap, ok := psVars.(map[string]string); ok {
			if charset, ok := psMap["charset"]; ok && charset != "" {
				return charset
			}
		}
	}
	return charsetFull
}
