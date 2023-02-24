# GraphQL


## Model generation
- Define or update models in `schema.graphqls`

- Regenerate the models by running the command:

```
$ go run github.com/99designs/gqlgen generate
```

- Query or Mutation functions will be stored in `schema.resolver.go`, update the logic if needed.