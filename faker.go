package restclient

import (
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


// substituteFakerVariables handles the substitution of faker/person data variables
func substituteFakerVariables(text string) string {
	text = substituteVSCodeStyleFakers(text)
	text = substituteJetBrainsStyleFakers(text)
	return text
}

// substituteVSCodeStyleFakers handles VS Code style faker variables
func substituteVSCodeStyleFakers(text string) string {
	text = reRandomFirstName.ReplaceAllStringFunc(text, getRandomFirstName)
	text = reRandomLastName.ReplaceAllStringFunc(text, getRandomLastName)
	text = reRandomFullName.ReplaceAllStringFunc(text, getRandomFullName)
	text = reRandomJobTitle.ReplaceAllStringFunc(text, getRandomJobTitle)
	return text
}

// substituteJetBrainsStyleFakers handles JetBrains style faker variables
func substituteJetBrainsStyleFakers(text string) string {
	text = reRandomFirstNameDot.ReplaceAllStringFunc(text, getRandomFirstName)
	text = reRandomLastNameDot.ReplaceAllStringFunc(text, getRandomLastName)
	text = reRandomFullNameDot.ReplaceAllStringFunc(text, getRandomFullName)
	text = reRandomJobTitleDot.ReplaceAllStringFunc(text, getRandomJobTitle)
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