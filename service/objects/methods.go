package objects

import (
	"refugio/utils"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

/* PessoaResult validation and cleaning */
func (p *PessoaResult) Clean() *PessoaResult {
	p.Nome = cleanNome(p.Nome)
	p.Abrigo = cleanCommon(p.Abrigo)

	return p
}

func (p *PessoaResult) DeduplicateAbrigo(abrigosMapping map[string]string) *PessoaResult {
	if abrigoDedup, ok := abrigosMapping[strings.ToLower(p.Abrigo)]; ok {
		p.Abrigo = abrigoDedup
	}
	return p
}

func (p *PessoaResult) Validate() (bool, *PessoaResult) {
	if p.Nome == "" {
		return false, p
	}
	if p.Abrigo == "" {
		return false, p
	}
	if len(onlyLettersAndNumbers(p.Nome)) == 0 {
		return false, p
	}
	if len(onlyLettersAndNumbers(p.Abrigo)) == 0 {
		return false, p
	}
	return true, p
}

func (p *PessoaResult) AggregateKey() string {
	caser := cases.Lower(language.BrazilianPortuguese)
	return utils.RemoveAccents(caser.String(onlyLettersAndNumbers(p.Nome + p.Abrigo)))
}

func cleanNome(name string) string {
	caser := cases.Title(language.BrazilianPortuguese)

	// Basic cleaning
	name = cleanCommon(name)

	// Enforce title case
	name = caser.String(name)

	// Replace long numbers with an empty string. This is to remove sensitive information like phones and document numbers
	regexPhoneNumbers := regexp.MustCompile(`\d{3,}`)
	name = regexPhoneNumbers.ReplaceAllString(name, "")

	return name
}

func cleanCommon(str string) string {
	// Strip leading and trailing whitespace
	str = strings.TrimSpace(str)

	// Remove extra spaces
	str = utils.RemoveExtraSpaces(str)

	// Remove linebreaks, etc
	regexLineBreaks := regexp.MustCompile(`[\r\n\t]`)
	str = regexLineBreaks.ReplaceAllString(str, "")

	str = strings.Replace(str, "/", "", -1)
	return str
}

func onlyLettersAndNumbers(str string) string {
	re := regexp.MustCompile(`[^\p{L}\p{N}]`)

	result := re.ReplaceAllString(str, "")
	return result
}
