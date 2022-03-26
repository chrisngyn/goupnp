// Package envelope is responsible for encoding and decoding SOAP envelopes.
package envelope

import (
	"encoding/xml"
	"fmt"
	"io"
)

// FaultDetail carries XML-encoded application-specific Fault details.
type FaultDetail struct {
	Raw []byte `xml:",innerxml"`
}

// Fault implements error, and contains SOAP fault information.
type Fault struct {
	Code   string      `xml:"faultcode"`
	String string      `xml:"faultstring"`
	Actor  string      `xml:"faultactor"`
	Detail FaultDetail `xml:"detail"`
}

func (fe *Fault) Error() string {
	return fmt.Sprintf("SOAP fault code=%s: %s", fe.Code, fe.String)
}

// Various "constant" bytes used in the written envelope.
var (
	envOpen  = []byte(xml.Header + `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body>`)
	env1     = []byte(`<u:`)
	env2     = []byte(` xmlns:u="`)
	env3     = []byte(`">`)
	env4     = []byte(`</u:`)
	env5     = []byte(`>`)
	envClose = []byte(`</s:Body></s:Envelope>`)
)

// Action wraps a SOAP action to be read or written as part of a SOAP envelope.
type Action struct {
	// XMLName specifies the XML element namespace (URI) and name. Together
	// these identify the SOAP action.
	XMLName xml.Name
	// Args is an arbitrary struct containing fields for encoding or decoding
	// arguments. See https://pkg.go.dev/encoding/xml@go1.17.1#Marshal and
	// https://pkg.go.dev/encoding/xml@go1.17.1#Unmarshal for details on
	// annotating fields in the structure.
	Args any `xml:",any"`
}

// Write marshals a SOAP envelope to the writer. Errors can be from the writer
// or XML encoding.
func Write(w io.Writer, action *Action) error {
	// Experiments with one router have shown that it 500s for requests where
	// the outer default xmlns is set to the SOAP namespace, and then
	// reassigning the default namespace within that to the service namespace.
	// Most of the code in this function is hand-coding the outer XML to
	// workaround this.
	// Resolving https://github.com/golang/go/issues/9519 might remove the need
	// for this workaround.

	_, err := w.Write(envOpen)
	if err != nil {
		return err
	}
	_, err = w.Write(env1)
	if err != nil {
		return err
	}
	err = xml.EscapeText(w, []byte(action.XMLName.Local))
	if err != nil {
		return err
	}
	_, err = w.Write(env2)
	if err != nil {
		return err
	}
	err = xml.EscapeText(w, []byte(action.XMLName.Space))
	if err != nil {
		return err
	}
	_, err = w.Write(env3)
	if err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	err = enc.Encode(action.Args)
	if err != nil {
		return err
	}
	err = enc.Flush()
	if err != nil {
		return err
	}
	_, err = w.Write(env4)
	if err != nil {
		return err
	}
	xml.EscapeText(w, []byte(action.XMLName.Local))
	if err != nil {
		return err
	}
	_, err = w.Write(env5)
	if err != nil {
		return err
	}
	_, err = w.Write(envClose)
	return err
}

// Read unmarshals a SOAP envelope from the reader. Errors can either be from
// the reader, XML decoding, or a *Fault.
func Read(r io.Reader, action *Action) error {
	env := envelope{
		Body: body{
			Action: action,
		},
	}

	dec := xml.NewDecoder(r)
	err := dec.Decode(&env)
	if err != nil {
		return err
	}

	if env.Body.Fault != nil {
		return env.Body.Fault
	}

	return nil
}

type envelope struct {
	XMLName       xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	EncodingStyle string   `xml:"http://schemas.xmlsoap.org/soap/envelope/ encodingStyle,attr"`
	Body          body     `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
}

type body struct {
	Fault  *Fault  `xml:"Fault"`
	Action *Action `xml:",any"`
}