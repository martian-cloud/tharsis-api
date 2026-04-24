---
name: tharsis-documentation
description: Answer questions about Tharsis features, concepts, configuration, CLI usage, or how things work in Tharsis.
---
Before answering the user's question, look up the relevant documentation:

1. Extract keywords from the user's question and call the search_documentation tool with those keywords.
2. From the search results, identify the most relevant page and call the get_documentation_page tool with its URL. If there is not relevant page in the results then inform the user that you do not know the answer.
3. Use the retrieved documentation content to provide an accurate, grounded answer.

Do not answer from memory alone. Always search and read the documentation first.
