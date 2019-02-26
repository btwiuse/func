package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"text/template"

	"github.com/func/func/tools/structdoc/structdoc"
	"github.com/spf13/cobra"
)

// A Data struct is passed to the template
type Data struct {
	File    string                       // Filename passed to structdoc
	Doc     string                       // Doc for type
	Comment string                       // Trailing comment for type
	Fields  map[string][]structdoc.Field // Fields in type
	Data    map[string]string            // Additional data passed in flag
}

var cmd = &cobra.Command{
	Use:   "structdoc",
	Short: "structdoc generates docs from struct tags",
	Long: `structdoc generates documentation from struct tags

structtag \
	--file somefile.go
	--struct MyStruct
	--template template.txt
	--data greeting=hello
	--output generated.md
`,
	Run: func(cmd *cobra.Command, args []string) {
		flagFile, _ := cmd.Flags().GetString("file")
		flagType, _ := cmd.Flags().GetString("struct")
		flagTmpl, _ := cmd.Flags().GetString("template")
		flagOutp, _ := cmd.Flags().GetString("output")
		flagVars, _ := cmd.Flags().GetStringToString("data")

		f, err := os.Open(flagFile)
		if err != nil {
			log.Fatal(err)
		}

		ast, err := structdoc.Parse(f, flagType)
		if err != nil {
			log.Fatal(err)
		}

		if err := f.Close(); err != nil {
			log.Fatal(err)
		}

		src, err := ioutil.ReadFile(flagTmpl)
		if err != nil {
			log.Fatal(err)
		}
		tmpl, err := template.New("").Parse(string(src))
		if err != nil {
			log.Fatal(err)
		}

		data := Data{
			File:    flagFile,
			Doc:     ast.Doc,
			Comment: ast.Comment,
			Fields:  make(map[string][]structdoc.Field),
			Data:    flagVars,
		}

		for _, f := range ast.Fields {
			for _, t := range f.Tags {
				f.Name = t.Name // Use struct tag name instead of field name
				data.Fields[t.Key] = append(data.Fields[t.Key], f)
			}
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			log.Fatal(err)
		}

		if flagOutp != "" {
			if err := ioutil.WriteFile(flagOutp, buf.Bytes(), 0644); err != nil {
				log.Fatal(err)
			}
			return
		}

		if _, err := buf.WriteTo(os.Stdout); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	cmd.Flags().StringP("file", "f", "", "Target file to read")
	cmd.Flags().StringP("struct", "s", "", "Struct name")
	cmd.Flags().StringP("template", "t", "", "Template file")
	cmd.Flags().StringP("output", "o", "", "Output file. If omitted, print to stdout")
	cmd.Flags().StringToStringP("data", "d", map[string]string{}, "Custom `key=value` data to pass to template")

	_ = cmd.MarkFlagRequired("file")
	_ = cmd.MarkFlagRequired("struct")
	_ = cmd.MarkFlagRequired("template")
}

func main() {
	_ = cmd.Execute()
}
