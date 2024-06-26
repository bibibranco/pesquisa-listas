package sheetscraper

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"refugio/objects"
	"refugio/repository"
	"refugio/utils"
	"refugio/utils/cuckoo"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	Pessoa     = "Pessoa"
	Completo   = "COMPLETO"
	Incompleto = "INCOMPLETO"
)

type SheetsSource struct{}
type SourceData struct {
	Data  interface{} `json:"data,omitempty"`
	Error error       `json:"error,omitempty"`
}

func (ss *SheetsSource) Read(sheetID string, sheetRange string) (interface{}, []*sheets.Sheet, error) {
	serviceAccJSON := utils.GetServiceAccountJSON(os.Getenv("SHEETS_SERVICE_ACCOUNT_JSON"))
	srv, err := sheets.NewService(context.Background(), option.WithCredentialsJSON(serviceAccJSON))
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	spreadsheet, _ := srv.Spreadsheets.Get(sheetID).Do()

	resp, err := srv.Spreadsheets.Values.Get(sheetID, sheetRange).Do()
	if err != nil {
		return nil, nil, err
	}

	return resp.Values, spreadsheet.Sheets, nil
}

func Scrape(isDryRun bool) {
	ss := SheetsSource{}
	var serializedData []*objects.PessoaResult
	var serializedSources []*objects.Source
	filter, err := cuckoo.GetCuckooFilter(Pessoa)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting cuckoo filter: %v", err)
		return
	}

	if os.Getenv("ENVIRONMENT") == "local" && !isDryRun {
		log.Panicln("Cannot run in local environment without dry run")
		return
	}

	abrigoMap := getAbrigosMapping()

	for _, cfg := range Config {
		if cfg.id != "1ym1_GhBA47LhH97HhggICESiUbKSH-e2Oii1peh6QF0" { // Planilhão
			serializedSources = append(serializedSources, &objects.Source{
				Nome:    cfg.name,
				SheetId: cfg.id,
				URL:     "",
			})
		}

		for _, sheetRange := range cfg.sheetRanges {
			content, tabs, err := ss.Read(cfg.id, sheetRange)

			seenSheets := make(map[string]bool)

			for _, tab := range tabs {
				if _, ok := seenSheets[tab.Properties.Title]; !ok {
					seenSheets[tab.Properties.Title] = true
					serializedSources[len(serializedSources)-1].Sheets = append(serializedSources[len(serializedSources)-1].Sheets, tab.Properties.Title)
				}
			}

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading sheet %s: %v\n", cfg.id, err)
				continue
			}
			fmt.Fprintf(os.Stdout, "Scraping data from sheetId %s, range %s\n", cfg.id, sheetRange)
			sheetNameAndRange := cfg.id + sheetRange
			switch sheetNameAndRange {
			// Offsets e customizações pra cada planilha hardcoded por enquanto
			case "1-1q4c8Ns6M9noCEhQqBE6gy3FWUv-VQgeUO9c7szGIM" + "COLÉGIO ADVENTISTA DE CANOAS - CACN!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 2 || len(row) < 3 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Colégio Adventista de Canoas",
						Nome:   row[2].(string),
					}
					if len(row) > 4 {
						p.Idade = row[3].(string)
					} else {
						p.Idade = ""
					}
					if len(row) > 8 {
						p.Observacao = row[8].(string)
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
				}
			case "1Kw8_Tl4cE4_hrb2APfSlNRli7IxgBbwGXq9d7aNSTzE" + "Cadastro inicial!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 6 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Escola Aurélio Reis",
						Nome:   row[1].(string),
					}
					if row[2].(string) != "" {
						p.Idade = row[2].(string)
					} else {
						p.Idade = ""
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}

					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1--z2fbczdFT4RSoji7jXc2jDDU5HqWgAU93NuROBQ78" + "Lista dos Acolhidos em Gravataí ":
				for i, row := range content.([][]interface{}) {
					if i < 3 || len(row) < 8 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: row[7].(string),
						Nome:   row[0].(string),
						Idade:  row[1].(string),
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1--z2fbczdFT4RSoji7jXc2jDDU5HqWgAU93NuROBQ78" + "Queila!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 4 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: row[6].(string),
						Nome:   row[0].(string),
						Idade:  row[1].(string),
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}

					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}

			case "10OnXFy-8TtUr3gw9yvtWroI7Z1psXGjdyBA3KMQKstE" + "Planilha1!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "FAPA",
						Nome:   row[1].(string),
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}

					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}

			case "14WIowAKQo5o_FviBw_6hRxnzAclw5xTvHbUiQuU8qDw" + "Cadastro!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Escola Municipal Elyseu Paglioli",
						Nome:   row[0].(string),
					}
					if len(row) > 5 {
						p.Idade = row[5].(string)
					} else {
						p.Idade = ""
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}

					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}

			case cfg.id + "Alojados!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 13 || len(row) < 3 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: row[1].(string),
						Nome:   row[2].(string),
					}

					if len(row) > 3 {
						p.Idade = row[3].(string)
					} else {
						p.Idade = ""
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "CADASTRO_ABRIGADOS!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 3 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: row[1].(string),
						Nome:   row[2].(string),
						Idade:  "",
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "ALOJADOS x ABRIGOS!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 13 || len(row) < 4 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: row[2].(string),
						Nome:   row[3].(string),
						Idade:  "",
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1frgtJ9eK05OqsyLwOBiZ2Q6E7e4_pWyrb7fJioqfEMs" + "ATUALIZADO 06/05!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 4 || len(row) < 3 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: row[2].(string),
						Nome:   row[0].(string),
						Idade:  "",
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "ESCOLA ANDRÉ PUENTE!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 2 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Escola André Puente",
						Nome:   row[0].(string),
						Idade:  "",
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "EMEF WALTER PERACCHI DE BARCELLOS!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 2 || len(row) < 3 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "EMEF Walter Peracchi de Barcellos",
						Nome:   row[1].(string),
						Idade:  row[2].(string),
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "CACHOEIRINHA!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: row[1].(string),
						Nome:   row[0].(string),
						Idade:  "",
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "COLÉGIO MARIA AUXILIADORA!A1:ZZ":
				for _, row := range content.([][]interface{}) {
					p := objects.Pessoa{
						Abrigo: "Colégio Maria Auxiliadora",
						Nome:   row[0].(string),
						Idade:  "",
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "ULBRA - Prédio 14!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "ULBRA - Prédio 14",
						Nome:   row[0].(string),
					}

					if len(row) > 2 {
						p.Idade = row[1].(string)
					} else {
						p.Idade = ""
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "COLÉGIO MIGUEL LAMPERT!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Colégio Miguel Lampert",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "AMORJI!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Associação dos Moradores do Jardim Igara II - AMORJI",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "ESCOLA RONDONIA!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Escola Rondônia",
						Nome:   row[0].(string),
					}

					if len(row) > 2 {
						p.Idade = row[1].(string)
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "Escola Jacob Longoni!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Escola Jacob Longoni",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "COLÉGIO ESPÍRITO SANTO!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}

					var p objects.Pessoa
					p = objects.Pessoa{
						Abrigo: "Colégio Espirito Santo",
						Nome:   row[0].(string),
						Idade:  row[1].(string),
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "CLUBE DOS EMPREGADOS DA PETROBRÁS!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}

					split := strings.Split(row[0].(string), "\n")
					for _, s := range split {
						p := objects.Pessoa{
							Abrigo: "Clube dos Empregados da Petrobras",
							Nome:   s,
							Idade:  "",
						}

						if os.Getenv("ENVIRONMENT") == "local" {
							fmt.Fprintf(os.Stdout, "%+v\n", p)
						}
						serializedData = append(serializedData, &objects.PessoaResult{
							Pessoa:    &p,
							SheetId:   &cfg.id,
							Timestamp: time.Now(),
						})
					}
				}
			case cfg.id + "Colegio Guajuviras!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 2 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Colégio Guajuviras",
						Nome:   row[0].(string),
					}

					if len(row) > 2 {
						p.Idade = row[1].(string)
					} else {
						p.Idade = ""
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "CEL São José!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "CEL São José",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "CR BRASIL!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 3 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "CR Brasil",
						Nome:   row[2].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "CSSGAPA!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Associação de Suboficiais e Sargentos da Guarnição de Aeronáutica de Porto Alegre",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "CTG Brazão do Rio Grande!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "CTG Brazão do Rio Grande",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "CTG Seiva Nativa!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "CTG Seiva Nativa",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "EMEF ILDO!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "EMEF Ildo Meneghetti",
						Nome:   row[1].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "Escola Irmao pedro!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Escola Irmão Pedro",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "FENIX!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Abrigo Fenix",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "PARÓQUIA SANTA LUZIA!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}

					split := strings.Split(row[0].(string), "\n")
					for _, s := range split {
						p := objects.Pessoa{
							Abrigo: "Paróquia Santa Luzia",
							Nome:   s,
							Idade:  "",
						}

						if os.Getenv("ENVIRONMENT") == "local" {
							fmt.Fprintf(os.Stdout, "%+v\n", p)
						}
						serializedData = append(serializedData, &objects.PessoaResult{
							Pessoa:    &p,
							SheetId:   &cfg.id,
							Timestamp: time.Now(),
						})
					}
				}
			case cfg.id + "IFRS- Canoas!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}

					firstCell := row[0].(string)
					if strings.Contains(firstCell, "PESSOAS QUE SAIRAM") {
						break
					}

					nome := strings.Split(firstCell, ". ")
					if len(nome) < 2 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: "Instituto Federal (IFRS) - Canoas",
						Nome:   nome[1],
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "Igreja Redenção Nazario!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Igreja Redenção Nazário",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "MODULAR!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Modular",
						Nome:   row[0].(string),
					}
					if len(row) > 2 {
						p.Idade = row[1].(string)
					} else {
						p.Idade = ""
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "Paroquia NSRosário!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Paróquia Nossa Senhora do Rosário",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "pediatria HU!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 2 || len(row) < 1 {
						continue
					}
					nome := row[0].(string)
					split := strings.Split(nome, ", ")

					p := objects.Pessoa{
						Abrigo: "Pediatria - Hospital Universitário Canoas",
						Nome:   split[0],
						Idade:  strings.Trim(split[1], ","),
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "Rua Itu, 672!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Rua Itu, 672",
						Nome:   strings.TrimRight(row[0].(string), "-"),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "ULBRA!A1:ZZ":
				seen := make(map[string]bool)
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 3 {
						continue
					}

					var p objects.Pessoa
					if len(row) > 4 {
						p = objects.Pessoa{
							Abrigo: utils.RemoveExtraSpaces("Ulbra" + " " + utils.RemoveSubstringInsensitive(row[4].(string), "ulbra")),
							Nome:   row[2].(string),
							Idade:  "",
						}
					} else {
						p = objects.Pessoa{
							Abrigo: "Ulbra",
							Nome:   row[2].(string),
							Idade:  "",
						}
					}
					if _, ok := seen[p.Nome]; !ok {
						seen[p.Nome] = true
						serializedData = append(serializedData, &objects.PessoaResult{
							Pessoa:    &p,
							SheetId:   &cfg.id,
							Timestamp: time.Now(),
						})
						if os.Getenv("ENVIRONMENT") == "local" {
							fmt.Fprintf(os.Stdout, "%+v\n", p)
						}
					}
				}
			case cfg.id + "Unilasalle!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Unilasalle",
						Nome:   row[1].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1-1q4c8Ns6M9noCEhQqBE6gy3FWUv-VQgeUO9c7szGIM" + "SESI!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "SESI",
						Nome:   row[1].(string),
					}
					if len(row) > 2 {
						p.Idade = row[2].(string)
					} else {
						p.Idade = ""
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "PARÓQUIA SAO LUIS!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Paróquia São Luis",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1Gf78W5yY0Yiljg-E0rYqbRjxYmBPcG2BtfpGwFk-K5M" + "Página1!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 2 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: row[1].(string),
						Nome:   row[0].(string),
						Idade:  "",
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "ENCONTRADOS!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "ULBRA - CANOAS PRÉDIO 11",
						Nome:   row[0].(string),
						Idade:  row[2].(string),
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "CIEP!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 4 || len(row) < 2 {
						continue
					}
					nome := row[1].(string)
					if strings.Contains(nome, "MENOR DE 1") {
						break
					}
					p := objects.Pessoa{
						Abrigo: "CIEP",
						Nome:   nome,
					}

					if len(row) > 3 {
						p.Idade = row[3].(string)
					} else {
						p.Idade = ""
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1RGRoIzSFQaaJF1xZsJhQsMJxXnXWzfZfas29T_PefmY" + "SESI!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 4 || len(row) < 3 {
						continue
					}
					nome := row[2].(string)

					if strings.Contains(nome, "MENOR DE 1") {
						break
					}

					p := objects.Pessoa{
						Abrigo: "SESI",
						Nome:   nome,
					}

					if len(row) > 3 {
						p.Idade = row[3].(string)
					} else {
						p.Idade = ""
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "LIBERATO!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 4 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Liberato",
						Nome:   row[1].(string),
					}

					if len(row) > 5 {
						p.Idade = row[5].(string)
					} else {
						p.Idade = ""
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "SINODAL!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 4 || len(row) < 3 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Sinodal",
						Nome:   row[2].(string),
					}

					if len(row) > 3 {
						p.Idade = row[3].(string)
					} else {
						p.Idade = ""
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "PARQUE DO TRABALHADOR!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 4 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Parque do Trabalhador",
						Nome:   row[1].(string),
					}
					if len(row) > 5 {
						p.Idade = row[5].(string)
					} else {
						p.Idade = ""
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "FENAC II!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 2 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "FENAC",
						Nome:   row[1].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "GINÁSIO DA BRIGADA!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 2 || len(row) < 3 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Ginásio da Brigada, Novo Hamburgo",
						Nome:   row[2].(string),
					}

					if len(row) > 3 {
						p.Idade = row[3].(string)
					} else {
						p.Idade = ""
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "IGREJA NOSSA SENHORA DAS GRAÇAS DA RONDÔNIA!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 2 || len(row) < 3 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Igreja Nossa Senhora das Graças da Rondônia",
						Nome:   row[2].(string),
					}

					if len(row) > 3 {
						p.Idade = row[3].(string)
					} else {
						p.Idade = ""
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "COMUNIDADE SANTO ANTONIO!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 2 || len(row) < 3 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Igreja Santo Antônio - Bairro Liberdade",
						Nome:   row[2].(string),
					}

					if len(row) > 3 {
						p.Idade = row[3].(string)
					} else {
						p.Idade = ""
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "PIO XII!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 2 || len(row) < 3 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Pio XII",
						Nome:   row[2].(string),
					}

					if len(row) > 3 {
						p.Idade = row[3].(string)
					} else {
						p.Idade = ""
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "LISTA MULHERES!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Sem informação",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "IGREJA NOSSA SENHORA DAS GRAÇAS !A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 4 || len(row) < 3 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "IGREJA NOSSA SENHORA DAS GRAÇAS - NH",
						Nome:   row[2].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "NOME/ABRIGO!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: row[1].(string) + " Eldorado do Sul",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "05/05 PONTAL!A1:ZZ", cfg.id + "06/05 PONTAL!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Pontal do Estaleiro",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "05/05 GASÔMETRO (NÃO MEXER!)!A1:ZZ", cfg.id + "06/05 GASÔMETRO (NÃO MEXER!)!A1:ZZ", cfg.id + "04/05 GASÔMETRO (NÃO MEXER!)!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Gasômetro",
						Nome:   row[1].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}

			case "1yuzazWMydzJKUoBnElV1YTxSKLJsT4fSVHfyJBjLlAY" + "Lista Abrigados!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "SESC Protásio",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "Abrigados Lajeado!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 3 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: row[2].(string),
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1O4NqkxHvFDoziS_zClwIjGIAVAGbYkfHTRrM6ogySTo" + "Página1!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 3 || len(row) < 1 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "Venâncio Aires",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "Resgatados!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 5 {
						continue
					}
					var abrigo string
					abrigo = row[4].(string)
					if abrigo == "" {
						abrigo = "Cruzeiro do Sul"
					}

					p := objects.Pessoa{
						Abrigo: abrigo,
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1AaQLs2Dqc6lrYstyF8UGLrihCzRRLsy8rlIRixJQ7VU" + "Página1!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 3 || len(row) < 1 {
						continue
					}
					var p objects.Pessoa
					pattern := `[0-9]+`
					re := regexp.MustCompile(pattern)
					replacedStr := re.ReplaceAllString(row[0].(string), "")
					if len(replacedStr) > 0 {
						p = objects.Pessoa{
							Abrigo: "Linha Herval - Venâncio Aires",
							Nome:   replacedStr,
							Idade:  "",
						}
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1IVtSmKRFynQH9I9Cox93YxZe0uwKfjx_CYFzKE96its" + "Sheet1!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}
					var p objects.Pessoa
					pattern := `[0-9]+`
					re := regexp.MustCompile(pattern)
					replacedStr := re.ReplaceAllString(row[0].(string), "")
					if len(replacedStr) > 0 {
						p = objects.Pessoa{
							Nome:  replacedStr,
							Idade: "",
						}
						if row[1].(string) != "" {
							p.Abrigo = row[1].(string)
						} else {
							p.Abrigo = "Abrigo Coelhão"
						}
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "16X-68-x7My4u0WEfscL7t4YYw_Ebeco6gaLhE80Q8Wc" + "Página1!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 6 {
						continue
					}
					var p objects.Pessoa
					var abrigo string

					abrigo = row[5].(string)
					if abrigo == "" {
						abrigo = "Desconhecido"
					}
					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   row[2].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1wvtgK7ZO9KuJsFDI9syyPWmEyqYoKw2PKssmgfo_jCU" + "Form Responses 1!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 4 {
						continue
					}
					var p objects.Pessoa
					var abrigo string
					var nome string

					nome = row[1].(string)
					if strings.Contains(nome, ".") {
						nomeSplit := strings.Split(nome, ".")
						if len(nomeSplit) > 1 {
							nome = nomeSplit[1]
						}
					}

					abrigo = row[3].(string)
					if abrigo == "" {
						abrigo = "Desconhecido"
					}
					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   nome,
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1fH7OA5bnY5OLfY7Xis6bVQq12VIhS_VIyYYekPBr5NA" + "Respostas ao formulário 1!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 4 {
						continue
					}
					var p objects.Pessoa
					var abrigo string

					nome := row[1].(string)

					abrigo = row[4].(string)
					if abrigo == "" {
						abrigo = "Desconhecido"
					}
					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   nome,
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1T_yd-M6BG1qYdQKeMo2U_AffqRCxkExqpB39iQXig5s" + "ENCONTRADOS!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 4 {
						continue
					}
					var p objects.Pessoa
					var abrigo string

					nome := row[0].(string)

					abrigo = fmt.Sprintf("Ulbra Canoas - Prédio %s - Sala %s", row[3].(string), row[4].(string))

					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   nome,
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1eC6z6RPNNarLMSqVqU-FQOHopCKWCN4CFDn34uTYGcA" + "Página 1!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 1 || len(row) < 4 {
						continue
					}
					var p objects.Pessoa
					var abrigo string
					var nome string
					var idade string

					nome = row[0].(string)
					// reg, err := regexp.Compile("[^a-zA-Z\\s]+")
					// if err != nil {
					// 	log.Fatal(err)
					// }
					// nome = reg.ReplaceAllString(nome, "")
					if len(row) > 4 {
						idade = row[4].(string)
					} else {
						idade = ""
					}

					abrigo = row[1].(string)
					if abrigo == "" {
						abrigo = "Desconhecido"
					}
					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   nome,
						Idade:  idade,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1eC6z6RPNNarLMSqVqU-FQOHopCKWCN4CFDn34uTYGcA" + "Página2!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 1 || len(row) < 4 {
						continue
					}
					var p objects.Pessoa
					var abrigo string
					var nome string
					var idade string

					nome = row[0].(string)
					// reg, err := regexp.Compile("[^a-zA-Z\\s]+")
					// if err != nil {
					// 	log.Fatal(err)
					// }
					// nome = reg.ReplaceAllString(nome, "")
					if len(row) > 4 {
						idade = row[4].(string)
					} else {
						idade = ""
					}

					abrigo = row[1].(string)
					if abrigo == "" {
						abrigo = "Desconhecido"
					}
					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   nome,
						Idade:  idade,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1eC6z6RPNNarLMSqVqU-FQOHopCKWCN4CFDn34uTYGcA" + "Página 3!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 1 || len(row) < 4 {
						continue
					}
					var p objects.Pessoa
					var abrigo string
					var nome string
					var idade string

					nome = row[0].(string)
					// reg, err := regexp.Compile("[^a-zA-Z\\s]+")
					// if err != nil {
					// 	log.Fatal(err)
					// }
					// nome = reg.ReplaceAllString(nome, "")
					if len(row) > 4 {
						idade = row[4].(string)
					} else {
						idade = ""
					}

					abrigo = row[1].(string)
					if abrigo == "" {
						abrigo = "Desconhecido"
					}
					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   nome,
						Idade:  idade,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1eC6z6RPNNarLMSqVqU-FQOHopCKWCN4CFDn34uTYGcA" + "Página 4!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 1 || len(row) < 4 {
						continue
					}
					var p objects.Pessoa
					var abrigo string
					var nome string
					var idade string

					nome = row[0].(string)
					// reg, err := regexp.Compile("[^a-zA-Z\\s]+")
					// if err != nil {
					// 	log.Fatal(err)
					// }
					// nome = reg.ReplaceAllString(nome, "")
					if len(row) > 4 {
						idade = row[4].(string)
					} else {
						idade = ""
					}

					abrigo = row[1].(string)
					if abrigo == "" {
						abrigo = "Desconhecido"
					}
					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   nome,
						Idade:  idade,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1eC6z6RPNNarLMSqVqU-FQOHopCKWCN4CFDn34uTYGcA" + "Página 5!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 1 || len(row) < 4 {
						continue
					}
					var p objects.Pessoa
					var abrigo string
					var nome string
					var idade string

					nome = row[0].(string)
					// reg, err := regexp.Compile("[^a-zA-Z\\s]+")
					// if err != nil {
					// 	log.Fatal(err)
					// }
					// nome = reg.ReplaceAllString(nome, "")
					if len(row) > 4 {
						idade = row[4].(string)
					} else {
						idade = ""
					}

					abrigo = row[1].(string)
					if abrigo == "" {
						abrigo = "Desconhecido"
					}
					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   nome,
						Idade:  idade,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1LdM2ZvYBNdtKekLgHPRs6lg9VGpD-7wBSZsE5c5Mptk" + "Página1!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 1 || len(row) < 3 {
						continue
					}
					var p objects.Pessoa
					var abrigo string

					abrigo = row[1].(string)
					if abrigo == "" {
						abrigo = "Desconhecido"
					}
					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1-cA0MB_1aQTOtXVL2pyPWSXjuTMg6U1PsyBAICjdGxo" + "Gravataí!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 1 || len(row) < 5 {
						continue
					}
					var p objects.Pessoa
					var abrigo string
					var nome string
					var idade string

					nome = row[0].(string)

					if len(row) > 4 {
						idade = row[4].(string)
					} else {
						idade = ""
					}

					abrigo = row[1].(string)
					if abrigo == "" {
						abrigo = "Desconhecido"
					}
					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   nome,
						Idade:  idade,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "16rN5pniNiIsbJAv25A0AfW5SdccJjPVDov7EDqwDOQM" + "Abrigados!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 1 || len(row) < 3 {
						continue
					}
					var p objects.Pessoa
					var abrigo string

					nome := row[0].(string)

					abrigo = row[2].(string)
					if abrigo == "" {
						abrigo = "Desconhecido"
					}
					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   nome,
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1gfQ28EPN99LQaZqZzMeB-pdxgK9SST1OYy-jTOl7rdk" + "Página1!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 1 || len(row) < 3 {
						continue
					}
					var p objects.Pessoa
					var abrigo string

					nome := row[0].(string)
					// reg, err := regexp.Compile("[^a-zA-Z\\s]+")
					// if err != nil {
					// 	log.Fatal(err)
					// }
					// nome = reg.ReplaceAllString(nome, "")

					abrigo = row[1].(string)
					if abrigo == "" {
						abrigo = "Desconhecido"
					}
					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   nome,
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1KgPjNIDQOmDA59A8u4HIOzsL41ZGQH97n-2jl99tfuU" + "Sheet1!A1:ZZ":
				for i, row := range content.([][]interface{}) {

					if i < 1 || len(row) < 3 {
						continue
					}
					var p objects.Pessoa
					var abrigo string

					nome := row[0].(string)
					// reg, err := regexp.Compile("[^a-zA-Z\\s]+")
					// if err != nil {
					// 	log.Fatal(err)
					// }
					// nome = reg.ReplaceAllString(nome, "")

					abrigo = row[1].(string)
					if abrigo == "" {
						abrigo = "Desconhecido"
					}
					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   nome,
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "Giovana!A1:ZZ", cfg.id + "IA!A1:ZZ", cfg.id + "Lidia!A1:ZZ", cfg.id + "Lari!A1:ZZ", cfg.id + "Fernanda Auricchio!A1:ZZ", cfg.id + "Sylvia!A1:ZZ", cfg.id + "Lorena!A1:ZZ", cfg.id + "Raquel!A1:ZZ", cfg.id + "Bruna Oliveira!A1:ZZ", cfg.id + "Vania!A1:ZZ", cfg.id + "Nicole Silva!A1:ZZ", cfg.id + "Voluntário x!A1:ZZ", cfg.id + "Karina!A1:ZZ", cfg.id + "Teresa!A1:ZZ", cfg.id + "Stéfani!A1:ZZ", cfg.id + "Maya!A1:ZZ", cfg.id + "Rhana!A1:ZZ", cfg.id + "Bruna!A1:ZZ", cfg.id + "Luan!A1:ZZ", cfg.id + "Daniel!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 2 || len(row) < 5 {
						continue
					}
					var p objects.Pessoa
					var abrigo string

					abrigo = row[4].(string)
					if abrigo == "" {
						abrigo = "Desconhecido"
					}
					p = objects.Pessoa{
						Abrigo: abrigo,
						Idade:  "",
					}
					pattern := `\d+\.\s+([A-ZÁÉÍÓÚÂÊÎÔÛÃÕÄËÏÖÜÀÈÌÒÙÇ\s]+)\s+-\s+BL`
					re := regexp.MustCompile(pattern)
					match := re.FindStringSubmatch(row[0].(string))

					if len(match) > 1 {
						p.Nome = match[1]
					} else {
						p.Nome = row[0].(string)
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "Caio!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 2 || len(row) < 5 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: row[4].(string),
						Idade:  "",
						Nome:   row[1].(string),
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case cfg.id + "Matheus!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 3 || len(row) < 2 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: row[2].(string),
						Nome:   row[1].(string),
					}
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1xaEPlk8JonATIOAvQEc0Dev-QVAzx2AwUzLHBhbA3rI" + "Página1!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 2 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: "Viaduto Santa Rita - Eldorado",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1FRHLIpLOE0xr7IwecZHU6Q6QMkescPuqjtxmjIb2GI8" + "AD55!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) <= 2 {
						continue
					}
					var p objects.Pessoa
					var idade string

					if len(row) > 1 {
						idade = row[1].(string)
					} else {
						idade = ""
					}

					p = objects.Pessoa{
						Abrigo: "Assembléia de Deus 55",
						Nome:   row[0].(string),
						Idade:  idade,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1FRHLIpLOE0xr7IwecZHU6Q6QMkescPuqjtxmjIb2GI8" + "CESE!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}
					var p objects.Pessoa
					var idade string

					if len(row) > 1 {
						idade = row[1].(string)
					} else {
						idade = ""
					}

					p = objects.Pessoa{
						Abrigo: "Comunidade Evangélica Semear Esperança",
						Nome:   row[0].(string),
						Idade:  idade,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1FRHLIpLOE0xr7IwecZHU6Q6QMkescPuqjtxmjIb2GI8" + "Comunidade Santa Clara!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 1 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: "Comunidade Santa Clara",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1FRHLIpLOE0xr7IwecZHU6Q6QMkescPuqjtxmjIb2GI8" + "CTG Guapos da Amizade!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 3 {
						continue
					}
					var idade string

					if len(row) > 1 {
						idade = row[1].(string)
					} else {
						idade = ""
					}

					p := objects.Pessoa{
						Abrigo: "CTG Guapos da Amizade",
						Nome:   row[0].(string),
						Idade:  idade,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1FRHLIpLOE0xr7IwecZHU6Q6QMkescPuqjtxmjIb2GI8" + "Gaditas!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 4 || len(row) < 1 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: "Associação Gaditas",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1FRHLIpLOE0xr7IwecZHU6Q6QMkescPuqjtxmjIb2GI8" + "Ginásio Placar!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 4 || len(row) < 1 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: "Ginásio Placar",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1FRHLIpLOE0xr7IwecZHU6Q6QMkescPuqjtxmjIb2GI8" + "ONG Vida Viva!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 3 || len(row) < 1 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: "ONG Vida Viva",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1FRHLIpLOE0xr7IwecZHU6Q6QMkescPuqjtxmjIb2GI8" + "Onze Unidos!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 4 || len(row) < 1 {
						continue
					}
					var p objects.Pessoa

					var data string
					var name string
					var idade string
					var observacao string

					data = row[0].(string)

					splitVirgula := strings.Split(data, ",")
					name = splitVirgula[0]
					if len(splitVirgula) > 1 {
						idade = strings.Split(splitVirgula[1], " - ")[0]
					} else {
						idade = ""
					}
					splitHifen := strings.Split(data, "-")
					if len(splitHifen) > 1 {
						observacao = splitHifen[1]
					} else {
						observacao = ""
					}

					p = objects.Pessoa{
						Abrigo:     "Onze Unidos",
						Nome:       name,
						Idade:      idade,
						Observacao: observacao,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1FRHLIpLOE0xr7IwecZHU6Q6QMkescPuqjtxmjIb2GI8" + "CTG Carreteiros!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 4 || len(row) < 1 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: "CTG Carreteiros",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1FRHLIpLOE0xr7IwecZHU6Q6QMkescPuqjtxmjIb2GI8" + "Abrigo Santa Clara!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 4 || len(row) < 1 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: "Abrigo Santa Clara",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1FRHLIpLOE0xr7IwecZHU6Q6QMkescPuqjtxmjIb2GI8" + "SESI!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 4 || len(row) < 1 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: "SESI Cachoeirinha",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1FRHLIpLOE0xr7IwecZHU6Q6QMkescPuqjtxmjIb2GI8" + "Paróquia Santa Luzia (bairro Fátima)!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 4 || len(row) < 1 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: "Paróquia Santa Luzia - Cachoeirinha",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1FRHLIpLOE0xr7IwecZHU6Q6QMkescPuqjtxmjIb2GI8" + "Igreja Betel!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 4 || len(row) < 1 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: "Igreja Betel - Cachoeirinha",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1TVv1WEjrPBpnKsFIV60jz0kWPK6idovmnJDaGg6KKXw" + "Página1!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 2 || len(row) < 4 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: "SESI",
						Nome:   row[0].(string),
						Idade:  row[3].(string),
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			case "1kKfTi8N-XL2bcML8Xtf3cT1FNIzinqh4woHDjHn2Bgs" + "ATUALIZADO 05/05!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 4 || len(row) < 3 {
						continue
					}
					var p objects.Pessoa

					var observacao string

					if len(row) > 3 {
						observacao = row[3].(string)
					} else {
						observacao = ""
					}

					p = objects.Pessoa{
						Abrigo:     row[2].(string),
						Nome:       row[0].(string),
						Idade:      "",
						Observacao: observacao,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}

					sheetId := "1frgtJ9eK05OqsyLwOBiZ2Q6E7e4_pWyrb7fJioqfEMs"

					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &sheetId,
						Timestamp: time.Now(),
					})
				}
			case "1kKfTi8N-XL2bcML8Xtf3cT1FNIzinqh4woHDjHn2Bgs" + "ATUALIZADO 06/05!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 3 || len(row) < 3 {
						continue
					}
					var p objects.Pessoa

					var observacao string
					var abrigo string

					if len(row) > 2 {
						abrigo = row[2].(string)
					} else {
						abrigo = ""
					}

					if len(row) > 3 {
						observacao = row[3].(string)
					} else {
						observacao = ""
					}

					p = objects.Pessoa{
						Abrigo:     abrigo,
						Nome:       row[0].(string),
						Idade:      "",
						Observacao: observacao,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}

					sheetId := "1frgtJ9eK05OqsyLwOBiZ2Q6E7e4_pWyrb7fJioqfEMs"

					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &sheetId,
						Timestamp: time.Now(),
					})
				}
			case "1kKfTi8N-XL2bcML8Xtf3cT1FNIzinqh4woHDjHn2Bgs" + "ATUALIZADO 07/05!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 3 || len(row) < 3 {
						continue
					}
					var p objects.Pessoa

					var abrigo string

					if len(row) > 2 {
						abrigo = row[2].(string)
					} else {
						abrigo = ""
					}

					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}

					sheetId := "1frgtJ9eK05OqsyLwOBiZ2Q6E7e4_pWyrb7fJioqfEMs"

					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &sheetId,
						Timestamp: time.Now(),
					})
				}
			case "1kKfTi8N-XL2bcML8Xtf3cT1FNIzinqh4woHDjHn2Bgs" + "ATUALIZADO 08/05!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 2 || len(row) < 3 {
						continue
					}
					var p objects.Pessoa

					var observacao string
					var abrigo string

					if len(row) > 2 {
						abrigo = row[2].(string)
					} else {
						abrigo = ""
					}

					if len(row) > 3 {
						observacao = row[3].(string)
					} else {
						observacao = ""
					}

					p = objects.Pessoa{
						Abrigo:     abrigo,
						Nome:       row[0].(string),
						Idade:      "",
						Observacao: observacao,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					sheetId := "1frgtJ9eK05OqsyLwOBiZ2Q6E7e4_pWyrb7fJioqfEMs"
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &sheetId,
						Timestamp: time.Now(),
					})
				}
			case "1K3DRVlSpK3tWQ1B83Q9pxkhSivIsmf38FTb6SVjMzT4" + "Resgatados Prefeitura SL!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 3 {
						continue
					}
					var p objects.Pessoa
					var idade string

					if len(row) > 3 {
						idade = row[3].(string)
					} else {
						idade = ""
					}

					p = objects.Pessoa{
						Abrigo: row[0].(string),
						Nome:   row[1].(string),
						Idade:  idade,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					sheetId := "1frgtJ9eK05OqsyLwOBiZ2Q6E7e4_pWyrb7fJioqfEMs"
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &sheetId,
						Timestamp: time.Now(),
					})
				}
			case "1K3DRVlSpK3tWQ1B83Q9pxkhSivIsmf38FTb6SVjMzT4" + "RESGATADOS/ABRIGADOS!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 2 {
						continue
					}
					var p objects.Pessoa

					var abrigo string
					var observacao string

					if len(row) > 2 && row[2] != "" {
						abrigo = row[2].(string)
					} else {
						abrigo = "Desconhecido"
					}

					if len(row) > 3 {
						observacao = row[3].(string)
					} else {
						observacao = ""
					}

					p = objects.Pessoa{
						Abrigo:     abrigo,
						Nome:       row[0].(string),
						Idade:      "",
						Observacao: observacao,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					sheetId := "1frgtJ9eK05OqsyLwOBiZ2Q6E7e4_pWyrb7fJioqfEMs"
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &sheetId,
						Timestamp: time.Now(),
					})
				}
			case "1K3DRVlSpK3tWQ1B83Q9pxkhSivIsmf38FTb6SVjMzT4" + "Resgatados - Fernanda!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 0 || len(row) < 2 {
						continue
					}
					var p objects.Pessoa

					var observacao string

					if len(row) > 3 {
						observacao = row[3].(string)
					} else {
						observacao = ""
					}

					p = objects.Pessoa{
						Abrigo:     row[2].(string),
						Nome:       row[0].(string),
						Idade:      "",
						Observacao: observacao,
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					sheetId := "1frgtJ9eK05OqsyLwOBiZ2Q6E7e4_pWyrb7fJioqfEMs"
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &sheetId,
						Timestamp: time.Now(),
					})
				}
			case "1TvBXpT1vZpuAffc2rb8VE2mBMEFnG1_sqIlIL4b1PuA" + "Velha Cambona!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 4 || len(row) < 1 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: "Velha Cambona",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					sheetId := "1TvBXpT1vZpuAffc2rb8VE2mBMEFnG1_sqIlIL4b1PuA"
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &sheetId,
						Timestamp: time.Now(),
					})
				}
			case "1TvBXpT1vZpuAffc2rb8VE2mBMEFnG1_sqIlIL4b1PuA" + "NSra Fátima!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 4 || len(row) < 1 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: "Nossa Sra. de Fátima",
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					sheetId := "1TvBXpT1vZpuAffc2rb8VE2mBMEFnG1_sqIlIL4b1PuA"
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &sheetId,
						Timestamp: time.Now(),
					})
				}
			case "1TvBXpT1vZpuAffc2rb8VE2mBMEFnG1_sqIlIL4b1PuA" + "Vila Rica!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 4 || len(row) < 5 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: "Vila Rica",
						Nome:   row[0].(string),
						Idade:  row[4].(string),
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					sheetId := "1TvBXpT1vZpuAffc2rb8VE2mBMEFnG1_sqIlIL4b1PuA"
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &sheetId,
						Timestamp: time.Now(),
					})
				}
			case "1q3Z2iX_vop9EumvB-4UyZsVQl58ZQ0M1JnwQsc6HAAo" + "06/05!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 2 || len(row) < 2 {
						continue
					}
					var abrigo string

					if len(row) > 4 && row[4] != "-" {
						abrigo = row[4].(string)
					} else {
						abrigo = "Desconhecido"
					}

					p := objects.Pessoa{
						Abrigo: abrigo,
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					sheetId := "1TvBXpT1vZpuAffc2rb8VE2mBMEFnG1_sqIlIL4b1PuA"
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &sheetId,
						Timestamp: time.Now(),
					})
				}
			case "1q3Z2iX_vop9EumvB-4UyZsVQl58ZQ0M1JnwQsc6HAAo" + "07/05!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 2 || len(row) < 2 {
						continue
					}

					p := objects.Pessoa{
						Abrigo: row[1].(string),
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					sheetId := "1TvBXpT1vZpuAffc2rb8VE2mBMEFnG1_sqIlIL4b1PuA"
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &sheetId,
						Timestamp: time.Now(),
					})
				}
			case "1q3Z2iX_vop9EumvB-4UyZsVQl58ZQ0M1JnwQsc6HAAo" + "08/05!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 2 || len(row) < 2 {
						continue
					}
					var abrigo string

					if len(row) > 1 && row[1] != "" {
						abrigo = row[1].(string)
					} else {
						abrigo = "Desconhecido"
					}

					p := objects.Pessoa{
						Abrigo: abrigo,
						Nome:   row[0].(string),
						Idade:  "",
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					sheetId := "1TvBXpT1vZpuAffc2rb8VE2mBMEFnG1_sqIlIL4b1PuA"
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &sheetId,
						Timestamp: time.Now(),
					})
				}
			case "1oMPwqFsfjlHB1snApt_BGGJrwTSmFn_R8_4Bm7ufAoY" + "Página1!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 5 {
						continue
					}

					if len(row) < 5 && row[4] == "" {
						continue
					}

					var p objects.Pessoa
					var abrigo string

					if len(row) > 3 && row[4] != "" {
						abrigo = row[4].(string)
					}

					p = objects.Pessoa{
						Abrigo: abrigo,
						Nome:   row[0].(string),
						Idade:  row[1].(string),
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}
					sheetId := "1TvBXpT1vZpuAffc2rb8VE2mBMEFnG1_sqIlIL4b1PuA"
					url := "https://wa.me/5554996016629"
					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &sheetId,
						URL:       &url,
						Timestamp: time.Now(),
					})
				}
			case "1ym1_GhBA47LhH97HhggICESiUbKSH-e2Oii1peh6QF0" + "Sheet1!A1:ZZ": // Planilhão
				for i, row := range content.([][]interface{}) {
					// Sem validações de comprimento de linha pq nós controlamos o conteúdo
					if i < 1 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: row[1].(string),
						Nome:   row[0].(string),
						Idade:  "",
					}

					sheetId := row[2].(string)
					url := row[3].(string)

					if row[5].(string) == Incompleto {
						continue
					}

					if len(row) > 6 && row[6].(string) != "" {
						source := objects.Source{
							SheetId: sheetId,
							URL:     url,
							Nome:    row[6].(string),
						}

						serializedSources = append(serializedSources, &source)
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}

					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &sheetId,
						URL:       &url,
						Timestamp: time.Now(),
					})
				}
			case "17GlFds1C-sdRdpWkZczzisTdItbdWgVAMXwXV60htyA" + "Página1!A1:ZZ":
				for i, row := range content.([][]interface{}) {
					if i < 1 || len(row) < 2 {
						continue
					}
					p := objects.Pessoa{
						Abrigo: "CESMAR",
						Nome:   row[1].(string),
						Idade:  "",
					}

					if len(row) > 6 {
						p.Observacao = row[6].(string)
					}

					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stdout, "%+v\n", p)
					}

					serializedData = append(serializedData, &objects.PessoaResult{
						Pessoa:    &p,
						SheetId:   &cfg.id,
						Timestamp: time.Now(),
					})
				}
			}

			var cleanedData []*objects.PessoaResult
			for _, pessoa := range serializedData {
				cleanPessoa := pessoa.Clean()
				pessoaWithDeduplicatedAbrigo := cleanPessoa.DeduplicateAbrigo(abrigoMap)
				isValid, validPessoa := pessoaWithDeduplicatedAbrigo.Validate()
				if !isValid {
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stderr, "Invalid PessoaResult data. Nome: %+v Abrigo: %+v\n", pessoa.Nome, pessoa.Abrigo)
					}
					continue
				}
				key := validPessoa.AggregateKey()

				if filter.Lookup([]byte(key)) {
					if os.Getenv("ENVIRONMENT") == "local" {
						fmt.Fprintf(os.Stderr, "Pessoa: key %+v found in cuckoo filter, skipping\n", key)
					}
					continue
				} else {
					filter.Insert([]byte(key))
				}

				cleanedData = append(cleanedData, validPessoa)
			}
			if !isDryRun {
				repository.AddPessoasToFirestore(cleanedData)
				repository.UpdateFilterOnFirestore(Pessoa, filter.Encode())
			}
			fmt.Fprintf(os.Stdout, "Scraped data from sheetId %s, range %s. %d results. %d results after cleanup. Dry run? %v", cfg.id, sheetRange, len(serializedData), len(cleanedData), isDryRun)
			// Clearing arrays for next iteration, I don't think this is strictly needed but just in case.
			serializedData = serializedData[:0]
			cleanedData = cleanedData[:0]
			fmt.Fprintln(os.Stdout, "")
		}
	}
	// Remove duplicate sources
	uniqueSources := []*objects.Source{}

	existingSources, _ := repository.FetchSourcesFromFirestore()

	seen := map[string]bool{}
	for _, source := range serializedSources {
		key := source.URL + source.SheetId

		filteredSources := make([]*objects.Source, 0)
		for _, dbSource := range existingSources {
			if dbSource.SheetId == source.SheetId {
				filteredSources = append(filteredSources, dbSource)
			}
		}

		var lenFilteredSources int

		if len(filteredSources) > 0 {
			lenFilteredSources = len(filteredSources[0].Sheets)
		} else {
			lenFilteredSources = 0
		}

		if len(source.Sheets) > lenFilteredSources {
			fmt.Println("A new sheet was added to the source")
			notifyNewTab(source.SheetId)
		}

		allSheets := source.Sheets

		source.Sheets = slices.Compact(allSheets)

		if _, ok := seen[key]; !ok {
			seen[key] = true
			uniqueSources = append(uniqueSources, source)
		}
	}

	if os.Getenv("ENVIRONMENT") == "local" {
		fmt.Fprintf(os.Stdout, "\nFound %d sources:\n", len(uniqueSources))

		for _, s := range uniqueSources {
			fmt.Fprintf(os.Stdout, "%+v\n", s)
		}
	}

	if !isDryRun {
		repository.AddSourcesToFirestore(uniqueSources)
	}
}

func notifyNewTab(sheetId string) {
	url := os.Getenv("DISCORD_SOURCES_WEBHOOK")
	content := fmt.Sprintf("A new tab was added to the sheet https://docs.google.com/spreadsheets/d/%s.", sheetId)
	data := []byte(fmt.Sprintf(`{"content":"%s"}`, content))
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending notification to Discord: %v\n", err)
	}
	defer resp.Body.Close()
}
