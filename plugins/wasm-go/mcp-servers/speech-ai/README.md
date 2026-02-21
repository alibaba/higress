# Speech AI MCP Server

Speech processing suite for AI agents. Pronunciation assessment, speech-to-text, and text-to-speech — all accessible as MCP tools through Higress.

## Features

- **Pronunciation Assessment**: Score English pronunciation at phoneme, word, and sentence levels (0-100). Exceeds human expert inter-annotator agreement.
- **Speech-to-Text**: Transcribe audio with word-level timestamps and confidence scores. Sub-300ms latency.
- **Text-to-Speech**: Generate natural speech with 12 English voices (American and British). Speed control 0.5x-2.0x.

## Getting Started

### 1. Get an API Key

Visit [brainiall.com](https://brainiall.com) or subscribe via [Azure Marketplace](https://azuremarketplace.microsoft.com).

### 2. Configure in Higress

Register your API key on the [mcp.higress.ai](https://mcp.higress.ai) interface to receive a generated SSE endpoint URL.

### 3. Connect Your MCP Client

Add the generated URL to your MCP client configuration:

```json
{
  "mcpServers": {
    "speech-ai": {
      "url": "https://mcp.higress.ai/speech-ai/your-token"
    }
  }
}
```

## Tool Reference

### assess_pronunciation

Score pronunciation of spoken audio against reference text.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `audio` | string | Yes | Base64-encoded audio (WAV, MP3, FLAC, OGG) |
| `text` | string | Yes | Reference text to score against |
| `format` | string | No | Audio format (default: "wav") |

**Returns:** Overall score (0-100), sentence score, confidence (0-1), word-level scores with phoneme breakdown.

### transcribe_audio

Transcribe spoken audio to text with word-level timestamps.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `audio` | string | Yes | Base64-encoded audio |

**Returns:** Transcribed text, audio duration, word-level timestamps.

### synthesize_speech

Generate speech from text with selectable voices.

**Parameters:**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `text` | string | Yes | Text to synthesize (max 5000 chars) |
| `voice` | string | No | Voice ID (default: "af_heart") |
| `speed` | number | No | Speed multiplier 0.5-2.0 (default: 1.0) |

**Returns:** WAV audio data.

### list_tts_voices

List all 12 available English voices with metadata.

### check_pronunciation_service / check_stt_service / check_tts_service

Health check endpoints for each service.

## Available Voices

| ID | Name | Gender | Accent |
|----|------|--------|--------|
| af_heart | Heart | Female | American |
| af_bella | Bella | Female | American |
| am_adam | Adam | Male | American |
| am_michael | Michael | Male | American |
| bf_emma | Emma | Female | British |
| bm_lewis | Lewis | Male | British |
| +6 more voices | | | |

## Support

- Email: fasuizu@brainiall.com
- Website: [brainiall.com](https://brainiall.com)
