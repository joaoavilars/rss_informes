package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	tmpFile    = "rss.tmp"
	updateLog  = "rss.upd"
	compareLog = "rss.compare"
)

type Item struct {
	XMLName xml.Name `xml:"item"`
	Title   string   `xml:"title"`
	Link    string   `xml:"link"`
	Desc    string   `xml:"description"`
	Guid    string   `xml:"guid"`
}

type Channel struct {
	XMLName       xml.Name `xml:"channel"`
	Title         string   `xml:"title"`
	Link          string   `xml:"link"`
	Desc          string   `xml:"description"`
	LastBuildDate string   `xml:"lastBuildDate"`
	Items         []Item   `xml:"item"`
}

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Uso: ./rss_informes <arquivo_de_saida.xml>")
		return
	}

	xmlFile := os.Args[1]

	if _, err := os.Stat(updateLog); os.IsNotExist(err) {
		if _, err := os.Create(updateLog); err != nil {
			log.Fatal(err)
		}
	}

	if _, err := os.Stat(compareLog); os.IsNotExist(err) {
		if _, err := os.Create(compareLog); err != nil {
			log.Fatal(err)
		}
	}

	if err := processRSS(xmlFile); err != nil {
		log.Fatal(err)
	}
}

func processRSS(xmlFile string) error {
	url := "https://www.nfe.fazenda.gov.br/portal/informe.aspx?ehCTG=false"
	fmt.Println("Obtendo conteúdo HTML da página:", url)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	// Baixar conteúdo do site para o arquivo temporário
	tmpItems := make([]Item, 0)
	doc.Find(".conteudo .container .divInforme").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find("p").Text())
		link := url
		desc := strings.TrimSpace(s.Text())

		if title != "" && desc != "" {
			tmpItems = append(tmpItems, Item{
				Title: title,
				Link:  link,
				Desc:  desc,
			})
		}
	})

	// Ler itens do arquivo final
	finalItems, err := readItemsFromXML(xmlFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Criar ou carregar mapa de comparação de descrições
	compareMap := make(map[string]string)
	if len(finalItems) > 0 {
		for _, item := range finalItems {
			compareMap[item.Title] = item.Desc
		}
	} else {
		// Se o arquivo final estiver vazio, criar GUIDs iniciais
		for _, item := range tmpItems {
			item.Guid = generateGUID()
			finalItems = append(finalItems, item)
			compareMap[item.Title] = item.Desc
		}
		// Escrever itens iniciais no arquivo final
		if err := writeItemsToXML(xmlFile, finalItems); err != nil {
			return err
		}
	}

	// Comparar descrições e atualizar GUIDs conforme necessário
	updateMap := make(map[string]string)
	for _, item := range tmpItems {
		if prevDesc, ok := compareMap[item.Title]; ok {
			if item.Desc != prevDesc {
				item.Guid = generateGUID()
				updateMap[item.Title] = item.Desc
			}
		} else {
			item.Guid = generateGUID()
			updateMap[item.Title] = item.Desc
		}
	}

	// Atualizar itens no arquivo final com GUIDs atualizados
	for i, item := range finalItems {
		if desc, ok := updateMap[item.Title]; ok {
			finalItems[i].Desc = desc
			finalItems[i].Guid = generateGUID()
		}
	}

	// Escrever itens atualizados no arquivo final
	if err := writeItemsToXML(xmlFile, finalItems); err != nil {
		return err
	}

	// Salvar mapa de comparação atualizado
	if err := saveUpdateLog(updateMap); err != nil {
		return err
	}

	return nil
}

func readItemsFromXML(filePath string) ([]Item, error) {
	xmlFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer xmlFile.Close()

	byteValue, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		return nil, err
	}

	var rss RSS
	if err := xml.Unmarshal(byteValue, &rss); err != nil {
		return nil, err
	}

	return rss.Channel.Items, nil
}

func saveUpdateLog(updateMap map[string]string) error {
	file, err := os.Create(updateLog)
	if err != nil {
		return err
	}
	defer file.Close()

	for title, guid := range updateMap {
		fmt.Fprintf(file, "%s %s\n", title, guid)
	}

	return nil
}

func writeRSS(filePath string, rss *RSS) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := xml.NewEncoder(file)
	encoder.Indent("", "    ")
	if err := encoder.Encode(rss); err != nil {
		return err
	}

	return nil
}

func writeItemsToXML(filePath string, items []Item) error {
	rss := &RSS{
		Version: "2.0",
		Channel: Channel{
			Title: "Informes do Sefaz - NFe",
			Link:  "https://www.nfe.fazenda.gov.br/portal/informe.aspx?ehCTG=false",
			Desc:  "Estatísticas da NF-e",
			Items: items,
		},
	}

	return writeRSS(filePath, rss)
}

func generateGUID() string {
	uuid := make([]byte, 16)
	if _, err := rand.Read(uuid); err != nil {
		return ""
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return hex.EncodeToString(uuid)
}
