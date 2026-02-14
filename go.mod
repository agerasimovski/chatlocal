module github.com/agerasimovski/chatlocal

go 1.25.5

replace github.com/agerasimovski/chatlocal/llmapi => ./llmapi/

require (
	github.com/agerasimovski/chatlocal/llmapi v0.0.0-00010101000000-000000000000 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/crypto v0.48.0 // indirect
)
