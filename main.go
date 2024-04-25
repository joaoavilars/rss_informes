package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/net/html"
)

const (
	updateLog  = "rss.upd"
	compareLog = "rss.compare"
	tempXML    = "temp.xml"
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

func processRSS(xmlFile string) error {
	url := "https://www.nfe.fazenda.gov.br/portal/informe.aspx?ehCTG=false&AspxAutoDetectCookieSupport=0"
	fmt.Println("Obtendo conteúdo HTML da página:", url)

	// Executar o comando wget para obter o código-fonte da página
	cmd := exec.Command("wget", "-qO-", url)
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	// Parsear o HTML da página
	doc, err := html.Parse(strings.NewReader(string(output)))
	if err != nil {
		return err
	}

	// Inicializar a estrutura RSS
	rss := RSS{
		Version: "2.0",
		Channel: Channel{
			Title: "Informes do Sefaz - NFe",
			Link:  url,
			Desc:  "Estatisticas da NF-e",
		},
	}

	// Extrair os itens do feed RSS
	extractItems(doc, &rss)

	// Converter a estrutura RSS em XML temporário
	tempXMLBytes, err := xml.MarshalIndent(rss, "", "    ")
	if err != nil {
		return err
	}

	// Escrever o XML temporário no arquivo
	if err := ioutil.WriteFile(tempXML, tempXMLBytes, 0644); err != nil {
		return err
	}

	// Comparar o XML temporário com o XML final e adicionar itens diferentes ao XML final
	if err := compareXML(xmlFile, tempXML); err != nil {
		return err
	}

	fmt.Printf("Arquivo XML gerado com sucesso: %s\n", xmlFile)

	return nil
}

func compareXML(xmlFile, tempXML string) error {
	// Ler os itens do XML final
	finalItems, err := readItemsFromXML(xmlFile)
	if err != nil {
		return err
	}

	// Ler os itens do XML temporário
	tempItems, err := readItemsFromXML(tempXML)
	if err != nil {
		return err
	}

	// Adicionar itens diferentes ao XML final (no máximo 10 itens)
	for _, item := range tempItems {
		if !containsItem(finalItems, item) && len(finalItems) < 10 {
			finalItems = append(finalItems, item)
		}
	}

	// Escrever os itens atualizados no XML final
	if err := writeItemsToXML(xmlFile, finalItems); err != nil {
		return err
	}

	return nil
}

func containsItem(items []Item, newItem Item) bool {
	for _, item := range items {
		if item.Title == newItem.Title && item.Desc == newItem.Desc {
			return true
		}
	}
	return false
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

func writeItemsToXML(filePath string, items []Item) error {
	rss := &RSS{
		Version: "2.0",
		Channel: Channel{
			Title: "Informes do Sefaz - NFe",
			Link:  "https://www.nfe.fazenda.gov.br/portal/informe.aspx?ehCTG=false&AspxAutoDetectCookieSupport=0",
			Desc:  "Estatísticas da NF-e",
			Items: items,
		},
	}

	// Adicionar GUID aos novos itens
	for i := range rss.Channel.Items {
		if rss.Channel.Items[i].Guid == "" {
			rss.Channel.Items[i].Guid = generateGUID()
		}
	}

	return writeRSS(filePath, rss)
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

func generateGUID() string {
	uuid := make([]byte, 16)
	if _, err := rand.Read(uuid); err != nil {
		return ""
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return hex.EncodeToString(uuid)
}

func extractItems(n *html.Node, rss *RSS) {
	if n.Type == html.ElementNode && n.Data == "div" && n.Attr != nil {
		for _, attr := range n.Attr {
			if attr.Key == "id" && attr.Val == "conteudoDinamico" {
				extractItemDiv(n, rss)
				return // Processa apenas esta div e retorna
			}
		}
	}
	// Percorre recursivamente os nós filhos
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractItems(c, rss)
	}
}

func extractItemDiv(n *html.Node, rss *RSS) {
	if n.Type == html.ElementNode && n.Data == "div" && n.Attr != nil {
		for _, attr := range n.Attr {
			if attr.Key == "class" && attr.Val == "divInforme" {
				extractItemContent(n, rss)
				return // Processa apenas esta div e retorna
			}
		}
	}
	// Percorre recursivamente os nós filhos
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractItemDiv(c, rss)
	}
}

func extractItemContent(n *html.Node, rss *RSS) {
	var title, desc string

	// Extrair título
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "p" {
			title = strings.TrimSpace(c.FirstChild.Data)
			break
		}
	}

	// Extrair descrição
	var descBuilder strings.Builder
	var foundTitle bool
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if foundTitle && c.Type == html.ElementNode && c.Data == "div" {
			descBuilder.WriteString(strings.TrimSpace(c.FirstChild.Data))
			descBuilder.WriteString("\n")
		}
		if !foundTitle && c.Type == html.ElementNode && c.Data == "p" && strings.TrimSpace(c.FirstChild.Data) == title {
			foundTitle = true
		}
		if foundTitle && c.Type == html.TextNode {
			descBuilder.WriteString(strings.TrimSpace(c.Data))
			descBuilder.WriteString("\n")
		}
	}

	desc = descBuilder.String()

	// Adicionar o item ao RSS
	if title != "" && desc != "" {
		rss.Channel.Items = append(rss.Channel.Items, Item{
			Title: title,
			Desc:  desc,
		})
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Uso: ./nome_do_executável <arquivo_de_saida.xml>")
		return
	}

	xmlFile := os.Args[1]
	if err := processRSS(xmlFile); err != nil {
		log.Fatal(err)
	}
}
