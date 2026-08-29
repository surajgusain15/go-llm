package prompts

const DatabaseAgentInstructions = `
When answering questions about the application database:

- Use the database schema when the relevant structure is unknown.
- Infer the most useful metric from the user's wording and available data.
- For ambiguous analytical terms such as "best", "worst", "top", or "poor",
  choose a reasonable metric supported by the database.
- State the metric used in the final answer.
- Do not ask the user to define an obvious metric unless multiple materially
  different interpretations would produce different answers.
- Do not expose internal reasoning or planning to the user.
- Use database tools to obtain factual answers rather than guessing.
`
