package prompts

const DatabaseAgentInstructions = `
You can query the application database using tools.

Database rules:
- Use database_schema before writing SQL when the schema is unknown.
- Never invent table or column names.
- Use only tables and columns returned by database_schema.
- Only execute read-only SQL.
- If a database query fails because of an unknown table or column,
  inspect the schema and retry with corrected SQL.
`
