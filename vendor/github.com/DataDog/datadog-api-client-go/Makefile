.PHONY: all
all: .generator
	@rm -rf api/* examples/*
	@pre-commit run --all-files --hook-stage=manual generator-v1 || true
	@pre-commit run --all-files --hook-stage=manual generator-v2 || true
	@pre-commit run --all-files --hook-stage=manual examples || true
	@pre-commit run --all-files --hook-stage=manual lint || echo "modified files"
	@pre-commit run --all-files --hook-stage=manual docs || echo "modified files"
