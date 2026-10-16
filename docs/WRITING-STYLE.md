# Writing style

This document tells a writer how to write for this repository.

The style is [ASD-STE100][ste] Simplified Technical English, Issue 9, dated
2025-01-15. The style shortens the sentences and reduces the ambiguity.

> **This is not a certification.** No tool and no language model can certify
> text against ASD-STE100. The ASD Simplified Technical English Management
> Group does not certify software tools. The documents in this repository follow
> the writing style. They are not certified against the standard.

[ste]: https://www.asd-ste100.org/

## When to use the style

Use the style for these documents:

- `README.md`
- `docs/ARCHITECTURE.md`
- `docs/DETECTION.md`
- `docs/IOC.md`
- `docs/GLOSSARY.md`
- the comments that start with a function name
- the messages that the programs print

Do not use the style for these documents:

- `docs/THREAT_MODEL.md`. The hedging in a risk analysis **is** the information.
- `CODE_OF_CONDUCT.md`. The tone matters more than the sentence length.
- a commit message that explains a judgement call

## The rules that a checker can verify

A writer applies these rules in every document.

| Rule | Requirement |
|---|---|
| 4.1 | Write short sentences with a clear structure. |
| 5.1 | Use 20 words at most in a procedural sentence. |
| 6.3 | Use 25 words at most in a descriptive sentence. |
| 6.6 | Use 6 sentences at most in a paragraph. |
| 6.5 | Give each paragraph one topic. |
| 2.1 | Write multi-word nouns of 3 words at most. |
| 3.2 | Use only the infinitive, the imperative, the simple present, the simple past, the simple future, and the past participle as an adjective. |
| 3.5 | Use the "-ing" form only as a technical noun or inside a technical noun. |
| 3.6 | Use the active voice. Use the passive voice in a description only when the agent is unknown. |
| 5.2 | Write one instruction in each sentence. |
| 5.4 | Put the condition first, then a comma, then the instruction. |
| 4.2 | Do not omit words. Do not use contractions. |
| 8.1 | Do not use a semicolon. |
| 4.3 | Use a vertical list for a complex sequence. |
| 1.11 | Do not use two technical nouns for one item. |
| 7.1 | Start a safety instruction with `WARNING` or `CAUTION`. |
| 7.2 | Start a safety instruction with a clear instruction or condition. |
| 7.3 | Explain the risk after the instruction. |

## The limits for word count

The standard counts some elements as one word:

- a number
- a number with a unit, such as `50 psi`
- an abbreviation
- an alphanumeric identifier
- quoted text
- a title or a heading
- a proper name
- text in parentheses (rule 8.5)
- a hyphenated word (rule 8.7)

A writer counts these before a writer decides that a sentence is too long.

## The dictionary rules

The standard has 875 approved words and 1274 words that are not approved. The
writer must check each word in the official dictionary. The writer cannot check
875 words reliably by memory.

The rules that catch the most errors in practice:

| Use | Do not use |
|---|---|
| `APPROXIMATELY (adv)` | `ABOUT (prep)` for a value that is not exact |
| `CHECK (n)` with `do a check of` | `CHECK (v)` |
| `TEST (n)` with `DO THE TEST` | `TEST (v)` |
| `CAN` | `MAY`, `MIGHT`, `COULD` |
| `MUST` | `SHALL`, `SHOULD` |
| `MAKE SURE` | `ENSURE` |
| `START` | `INITIATE`, `COMMENCE`, `BEGIN` |
| `OBEY` | `FOLLOW` for obedience |
| `AT THIS TIME` | `NOW` for the present moment |

`CHECK`, `TEST`, and `VERIFY` are **not** approved verbs. The list of 208
approved verbs has no entry for any of them.

## The terminology rule

`docs/GLOSSARY.md` defines one term for one concept. A writer uses the same
term in every document.

A writer must not use these words for a program of this project:

- `payload`, `implant`, `agent client` for the **Beacon**
- `server` alone for the **Team Server**
- `heartbeat`, `ping`, `poll` for a **check-in**
- `token`, `API key` for the **shared secret**
- `timeout` for the **time to live**
- `alias`, `fake name` for the **masked name**

A writer adds a new term to the glossary before the first use of a new concept.

## The check before a commit

A writer checks the document against this list.

1. Each sentence has 25 words at most, or 20 in a procedure.
2. Each sentence uses the active voice.
3. Each sentence uses a simple tense.
4. No sentence uses a semicolon.
5. No sentence uses a contraction.
6. Each paragraph has one topic and 6 sentences at most.
7. Each multi-word noun has 3 words at most.
8. Each warning uses `WARNING`, then an instruction, then the risk.
9. Each new technical noun is in `docs/GLOSSARY.md` with its category.
10. Each new word is checked in the official dictionary.

A writer does not claim that the document conforms to the standard. A writer
claims only that it follows the writing style.
