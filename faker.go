package restclient

import (
	"fmt"
	"math/rand"
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


// substituteFakerVariables handles the substitution of faker/person data variables
func substituteFakerVariables(text string) string {
	text = substituteVSCodeStyleFakers(text)
	text = substituteJetBrainsStyleFakers(text)
	return text
}

// substituteVSCodeStyleFakers handles VS Code style faker variables
func substituteVSCodeStyleFakers(text string) string {
	// Person data
	text = reRandomFirstName.ReplaceAllStringFunc(text, getRandomFirstName)
	text = reRandomLastName.ReplaceAllStringFunc(text, getRandomLastName)
	text = reRandomFullName.ReplaceAllStringFunc(text, getRandomFullName)
	text = reRandomJobTitle.ReplaceAllStringFunc(text, getRandomJobTitle)
	
	// Contact data
	text = reRandomPhoneNumber.ReplaceAllStringFunc(text, getRandomPhoneNumber)
	text = reRandomStreetAddress.ReplaceAllStringFunc(text, getRandomStreetAddress)
	text = reRandomCity.ReplaceAllStringFunc(text, getRandomCity)
	text = reRandomState.ReplaceAllStringFunc(text, getRandomState)
	text = reRandomZipCode.ReplaceAllStringFunc(text, getRandomZipCode)
	text = reRandomCountry.ReplaceAllStringFunc(text, getRandomCountry)
	
	// Internet data
	text = reRandomUrl.ReplaceAllStringFunc(text, getRandomUrl)
	text = reRandomDomainName.ReplaceAllStringFunc(text, getRandomDomainName)
	text = reRandomUserAgent.ReplaceAllStringFunc(text, getRandomUserAgent)
	text = reRandomMacAddress.ReplaceAllStringFunc(text, getRandomMacAddress)
	
	return text
}

// substituteJetBrainsStyleFakers handles JetBrains style faker variables
func substituteJetBrainsStyleFakers(text string) string {
	// Person data
	text = reRandomFirstNameDot.ReplaceAllStringFunc(text, getRandomFirstName)
	text = reRandomLastNameDot.ReplaceAllStringFunc(text, getRandomLastName)
	text = reRandomFullNameDot.ReplaceAllStringFunc(text, getRandomFullName)
	text = reRandomJobTitleDot.ReplaceAllStringFunc(text, getRandomJobTitle)
	
	// Contact data - JetBrains style
	text = reRandomPhoneNumberDot.ReplaceAllStringFunc(text, getRandomPhoneNumber)
	text = reRandomStreetAddressDot.ReplaceAllStringFunc(text, getRandomStreetAddress)
	text = reRandomCityDot.ReplaceAllStringFunc(text, getRandomCity)
	text = reRandomStateDot.ReplaceAllStringFunc(text, getRandomState)
	text = reRandomZipCodeDot.ReplaceAllStringFunc(text, getRandomZipCode)
	text = reRandomCountryDot.ReplaceAllStringFunc(text, getRandomCountry)
	
	// Internet data - JetBrains style
	text = reRandomUrlDot.ReplaceAllStringFunc(text, getRandomUrl)
	text = reRandomDomainNameDot.ReplaceAllStringFunc(text, getRandomDomainName)
	text = reRandomUserAgentDot.ReplaceAllStringFunc(text, getRandomUserAgent)
	text = reRandomMacAddressDot.ReplaceAllStringFunc(text, getRandomMacAddress)
	
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
	areaCode := rand.Intn(900) + 100   // 100-999
	exchange := rand.Intn(900) + 100   // 100-999
	number := rand.Intn(10000)         // 0000-9999
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