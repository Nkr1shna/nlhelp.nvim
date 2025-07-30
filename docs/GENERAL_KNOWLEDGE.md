# General Knowledge Database

This document explains how to use the new general knowledge database functionality in the nvim-smart-keybind-search project.

## Overview

The system now supports storing and searching non-keybinding knowledge alongside keybindings. This includes:

- Vim concepts and principles
- Best practices and tips
- Plugin documentation
- Troubleshooting guides
- Learning resources
- Advanced features and techniques

## Hugging Face Dataset

The system now uses the comprehensive [Neovim Help Dataset](https://huggingface.co/datasets/aegis-nvim/neovim-help) from Hugging Face, which contains:

- **6,530 chunks** of Neovim help documentation
- Complete coverage of all Neovim commands and features
- File-based organization (e.g., `runtime/doc/windows.txt`)
- Section-based chunks with detailed explanations
- Examples and cross-references

This dataset provides much more comprehensive coverage than the previous manually curated knowledge base, giving users access to the complete Neovim documentation ecosystem.

## Architecture

The system uses three separate ChromaDB collections:

1. **`vim_knowledge`** - Built-in vim keybindings
2. **`user_keybindings`** - User-defined keybindings
3. **`general_knowledge`** - Non-keybinding knowledge and concepts

## Building the Database

### 1. Fetch Hugging Face Dataset (Recommended)

The system now uses the comprehensive Neovim help dataset from Hugging Face:

```bash
# Fetch and process the Hugging Face dataset
./scripts/run_huggingface_fetch.sh

# Or run manually:
go run scripts/huggingface/fetch_dataset.go
```

This will:
- Fetch the complete Neovim help documentation from Hugging Face
- Process and categorize the data automatically
- Save processed data to `scripts/processed_huggingface_data.json`
- Store the data in ChromaDB

### 2. Build All Collections

After fetching the dataset, build the complete database:

```bash
go run scripts/build_database.go
```

This will:
- Create the `vim_knowledge` collection with built-in keybindings
- Create the `general_knowledge` collection with the Hugging Face dataset
- Initialize all collections

### 3. Alternative: Use Your Own Scraped Data

If you have your own scraped data, you can still use it:

```bash
go run scripts/populate_general_knowledge.go your_scraped_data.json
```

## Data Format

Your scraped data should be in JSON format with the following structure:

```json
[
  {
    "id": "unique_identifier",
    "title": "Knowledge Title",
    "content": "Detailed content describing the knowledge...",
    "category": "category_name",
    "tags": ["tag1", "tag2", "tag3"],
    "source": "source_name",
    "difficulty": "beginner|intermediate|advanced",
    "examples": ["example1", "example2"],
    "related": ["related_topic1", "related_topic2"],
    "metadata": {
      "url": "https://example.com",
      "last_updated": "2024-01-15"
    }
  }
]
```

### Required Fields

- `id`: Unique identifier for the knowledge item
- `title`: Short, descriptive title
- `content`: Detailed content that will be vectorized
- `category`: Category for grouping (e.g., "vim_concepts", "plugins", "troubleshooting")

### Optional Fields

- `tags`: Array of tags for better searchability
- `source`: Source of the knowledge (e.g., "vim_documentation", "community")
- `difficulty`: Skill level required ("beginner", "intermediate", "advanced")
- `examples`: Array of practical examples
- `related`: Array of related topics
- `metadata`: Additional key-value pairs

## Search Behavior

When users query the system:

1. **All Collections Search** (default): Searches all three collections and returns mixed results
2. **Keybindings Only**: Searches only keybinding collections

The system prioritizes results as follows:
1. User keybindings (highest priority)
2. Built-in keybindings (medium priority)
3. General knowledge (lowest priority, but still included)

## Context Building

The system now builds context from all collection types:

- **Keybindings**: Shows keys, commands, descriptions, and modes
- **General Knowledge**: Shows titles, content summaries, categories, and difficulty levels

## Example Queries

The system can now handle queries like:

- "How do I use vim modes?"
- "What are the best practices for vim plugins?"
- "How do I record and use macros?"
- "What should I do if I get stuck in insert mode?"

## Configuration

You can control search behavior in the RAG agent configuration:

```go
config := &rag.AgentConfig{
    SearchAllCollections: true, // Set to false for keybindings only
    MaxSearchResults: 10,
    SimilarityThreshold: 0.3,
}
```

## API Usage

The system automatically includes general knowledge in search results when `SearchAllCollections` is enabled. No changes are needed to your existing API calls.

## Troubleshooting

### Collection Not Found

If you get collection errors, ensure the database was built:

```bash
go run scripts/build_database.go
```

### No General Knowledge Results

Check that:
1. The general knowledge collection was populated
2. `SearchAllCollections` is enabled in the agent config
3. The query is relevant to the stored knowledge

### Performance Issues

If search is slow with large knowledge bases:
1. Reduce `MaxSearchResults` in the agent config
2. Increase `SimilarityThreshold` to filter out less relevant results
3. Consider splitting large knowledge bases into smaller, more focused collections

## Adding New Knowledge

To add new knowledge:

1. Format your data according to the JSON schema above
2. Use the populate script to add it to the database
3. The system will automatically include it in future searches

## Example: Adding Plugin Documentation

```json
{
  "id": "fugitive_plugin",
  "title": "Git Integration with Fugitive",
  "content": "Fugitive is a powerful Git wrapper for Vim. It provides Git commands, diff views, and commit interfaces directly within Vim. Use :Gstatus to see repository status, :Gdiff to view diffs, and :Gcommit to commit changes.",
  "category": "plugins",
  "tags": ["git", "fugitive", "version_control", "plugin"],
  "source": "plugin_documentation",
  "difficulty": "intermediate",
  "examples": [
    ":Gstatus - show repository status",
    ":Gdiff - view file differences",
    ":Gcommit - commit changes"
  ],
  "related": ["git", "version_control", "vim_plugins"]
}
```

This knowledge will be automatically included when users ask about Git integration or the Fugitive plugin. 