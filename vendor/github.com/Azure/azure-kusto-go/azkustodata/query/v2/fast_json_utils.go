package v2

import (
	"encoding/json"
	"github.com/Azure/azure-kusto-go/azkustodata/errors"
)

// assertToken asserts that the next token in the decoder is the expected token.
func assertToken(dec *json.Decoder, expected json.Token) error {
	t, err := dec.Token()
	if err != nil {
		return err
	}
	if t != expected {
		return errors.ES(errors.OpUnknown, errors.KInternal, "Expected %v, got %v", expected, t)
	}
	return nil
}

// assertStringProperty asserts that the next token in the decoder is a string property with the expected name and value.
func assertStringProperty(dec *json.Decoder, name string, value json.Token) error {
	if err := assertToken(dec, json.Token(name)); err != nil {
		return err
	}
	if err := assertToken(dec, value); err != nil {
		return err
	}
	return nil
}

// getStringProperty reads a string property from the decoder, validating the name and returning the value.
func getStringProperty(dec *json.Decoder, name string) (string, error) {
	if err := assertToken(dec, json.Token(name)); err != nil {
		return "", err
	}
	t, err := dec.Token()
	if err != nil {
		return "", err
	}
	if s, ok := t.(string); ok {
		return s, nil
	}
	return "", errors.ES(errors.OpUnknown, errors.KInternal, "Expected string, got %v", t)
}

// getIntProperty reads an int property from the decoder, validating the name and returning the value.
func getIntProperty(dec *json.Decoder, name string) (int, error) {
	if err := assertToken(dec, json.Token(name)); err != nil {
		return 0, err
	}
	t, err := dec.Token()
	if err != nil {
		return 0, err
	}
	if s, ok := t.(json.Number); ok {
		i, err := s.Int64()
		if err != nil {
			return 0, err
		}
		return int(i), nil
	}
	return 0, errors.ES(errors.OpUnknown, errors.KInternal, "Expected string, got %v", t)
}
