---
name: recipe-hello-world
recipe_slug: hello-world
expected_status: completed
expected_output_contains: "Hello World"
gates: [not_empty, min_length_10]
---

Tests minimal recipe execution with single LLM step.
