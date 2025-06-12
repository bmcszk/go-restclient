package restclient

import (
	"math/rand"
	"regexp"
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

var (
	// Person/identity faker variables - VS Code style
	reRandomFirstName    = regexp.MustCompile(`{{\s*\$randomFirstName\s*}}`)
	reRandomLastName     = regexp.MustCompile(`{{\s*\$randomLastName\s*}}`)
	reRandomFullName     = regexp.MustCompile(`{{\s*\$randomFullName\s*}}`)
	reRandomJobTitle     = regexp.MustCompile(`{{\s*\$randomJobTitle\s*}}`)
	// Person/identity faker variables - JetBrains style
	reRandomFirstNameDot = regexp.MustCompile(`{{\s*\$random\.firstName\s*}}`)
	reRandomLastNameDot  = regexp.MustCompile(`{{\s*\$random\.lastName\s*}}`)
	reRandomFullNameDot  = regexp.MustCompile(`{{\s*\$random\.fullName\s*}}`)
	reRandomJobTitleDot  = regexp.MustCompile(`{{\s*\$random\.jobTitle\s*}}`)
)

// substituteFakerVariables handles the substitution of faker/person data variables
func substituteFakerVariables(text string) string {
	// Person/Identity data - VS Code style
	text = reRandomFirstName.ReplaceAllStringFunc(text, func(_ string) string {
		if len(firstNames) > 0 {
			return firstNames[rand.Intn(len(firstNames))]
		}
		return "John"
	})

	text = reRandomLastName.ReplaceAllStringFunc(text, func(_ string) string {
		if len(lastNames) > 0 {
			return lastNames[rand.Intn(len(lastNames))]
		}
		return "Doe"
	})

	text = reRandomFullName.ReplaceAllStringFunc(text, func(_ string) string {
		firstName := "John"
		lastName := "Doe"
		if len(firstNames) > 0 {
			firstName = firstNames[rand.Intn(len(firstNames))]
		}
		if len(lastNames) > 0 {
			lastName = lastNames[rand.Intn(len(lastNames))]
		}
		return firstName + " " + lastName
	})

	text = reRandomJobTitle.ReplaceAllStringFunc(text, func(_ string) string {
		if len(jobTitles) > 0 {
			return jobTitles[rand.Intn(len(jobTitles))]
		}
		return "Software Engineer"
	})

	// Person/Identity data - JetBrains style
	text = reRandomFirstNameDot.ReplaceAllStringFunc(text, func(_ string) string {
		if len(firstNames) > 0 {
			return firstNames[rand.Intn(len(firstNames))]
		}
		return "John"
	})

	text = reRandomLastNameDot.ReplaceAllStringFunc(text, func(_ string) string {
		if len(lastNames) > 0 {
			return lastNames[rand.Intn(len(lastNames))]
		}
		return "Doe"
	})

	text = reRandomFullNameDot.ReplaceAllStringFunc(text, func(_ string) string {
		firstName := "John"
		lastName := "Doe"
		if len(firstNames) > 0 {
			firstName = firstNames[rand.Intn(len(firstNames))]
		}
		if len(lastNames) > 0 {
			lastName = lastNames[rand.Intn(len(lastNames))]
		}
		return firstName + " " + lastName
	})

	text = reRandomJobTitleDot.ReplaceAllStringFunc(text, func(_ string) string {
		if len(jobTitles) > 0 {
			return jobTitles[rand.Intn(len(jobTitles))]
		}
		return "Software Engineer"
	})

	return text
}