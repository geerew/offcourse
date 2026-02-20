package models

import "fmt"

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	ASSET_METADATA_AUDIO_TABLE = "asset_metadata_audio"

	ASSET_METADATA_AUDIO_ASSET_ID       = "asset_id"
	ASSET_METADATA_AUDIO_LANGUAGE       = "language"
	ASSET_METADATA_AUDIO_CODEC          = "codec"
	ASSET_METADATA_AUDIO_PROFILE        = "profile"
	ASSET_METADATA_AUDIO_CHANNELS       = "channels"
	ASSET_METADATA_AUDIO_CHANNEL_LAYOUT = "channel_layout"
	ASSET_METADATA_AUDIO_SAMPLE_RATE    = "sample_rate"
	ASSET_METADATA_AUDIO_BIT_RATE       = "bit_rate"

	ASSET_METADATA_AUDIO_TABLE_ID             = ASSET_METADATA_AUDIO_TABLE + "." + BASE_ID
	ASSET_METADATA_AUDIO_TABLE_CREATED_AT     = ASSET_METADATA_AUDIO_TABLE + "." + BASE_CREATED_AT
	ASSET_METADATA_AUDIO_TABLE_UPDATED_AT     = ASSET_METADATA_AUDIO_TABLE + "." + BASE_UPDATED_AT
	ASSET_METADATA_AUDIO_TABLE_ASSET_ID       = ASSET_METADATA_AUDIO_TABLE + "." + ASSET_METADATA_AUDIO_ASSET_ID
	ASSET_METADATA_AUDIO_TABLE_LANGUAGE       = ASSET_METADATA_AUDIO_TABLE + "." + ASSET_METADATA_AUDIO_LANGUAGE
	ASSET_METADATA_AUDIO_TABLE_CODEC          = ASSET_METADATA_AUDIO_TABLE + "." + ASSET_METADATA_AUDIO_CODEC
	ASSET_METADATA_AUDIO_TABLE_PROFILE        = ASSET_METADATA_AUDIO_TABLE + "." + ASSET_METADATA_AUDIO_PROFILE
	ASSET_METADATA_AUDIO_TABLE_CHANNELS       = ASSET_METADATA_AUDIO_TABLE + "." + ASSET_METADATA_AUDIO_CHANNELS
	ASSET_METADATA_AUDIO_TABLE_CHANNEL_LAYOUT = ASSET_METADATA_AUDIO_TABLE + "." + ASSET_METADATA_AUDIO_CHANNEL_LAYOUT
	ASSET_METADATA_AUDIO_TABLE_SAMPLE_RATE    = ASSET_METADATA_AUDIO_TABLE + "." + ASSET_METADATA_AUDIO_SAMPLE_RATE
	ASSET_METADATA_AUDIO_TABLE_BIT_RATE       = ASSET_METADATA_AUDIO_TABLE + "." + ASSET_METADATA_AUDIO_BIT_RATE
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// AudioMetadata defines audio metadata for a video asset
type AudioMetadata struct {
	Base
	AssetID       string `db:"asset_id"`       // Immutable
	Language      string `db:"language"`       // Mutable
	Codec         string `db:"codec"`          // Mutable
	Profile       string `db:"profile"`        // Mutable
	Channels      int    `db:"channels"`       // Mutable
	ChannelLayout string `db:"channel_layout"` // Mutable
	SampleRate    int    `db:"sample_rate"`    // Mutable
	BitRate       int    `db:"bit_rate"`       // Mutable
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// AudioMetadataColumns returns the columns for use in a SELECT query
func AudioMetadataColumns() []string {
	return []string{
		fmt.Sprintf("%s AS %s", ASSET_METADATA_AUDIO_TABLE_ID, BASE_ID),
		fmt.Sprintf("%s AS %s", ASSET_METADATA_AUDIO_TABLE_CREATED_AT, BASE_CREATED_AT),
		fmt.Sprintf("%s AS %s", ASSET_METADATA_AUDIO_TABLE_UPDATED_AT, BASE_UPDATED_AT),
		fmt.Sprintf("%s AS %s", ASSET_METADATA_AUDIO_TABLE_ASSET_ID, ASSET_METADATA_AUDIO_ASSET_ID),
		fmt.Sprintf("%s AS %s", ASSET_METADATA_AUDIO_TABLE_LANGUAGE, ASSET_METADATA_AUDIO_LANGUAGE),
		fmt.Sprintf("%s AS %s", ASSET_METADATA_AUDIO_TABLE_CODEC, ASSET_METADATA_AUDIO_CODEC),
		fmt.Sprintf("%s AS %s", ASSET_METADATA_AUDIO_TABLE_PROFILE, ASSET_METADATA_AUDIO_PROFILE),
		fmt.Sprintf("%s AS %s", ASSET_METADATA_AUDIO_TABLE_CHANNELS, ASSET_METADATA_AUDIO_CHANNELS),
		fmt.Sprintf("%s AS %s", ASSET_METADATA_AUDIO_TABLE_CHANNEL_LAYOUT, ASSET_METADATA_AUDIO_CHANNEL_LAYOUT),
		fmt.Sprintf("%s AS %s", ASSET_METADATA_AUDIO_TABLE_SAMPLE_RATE, ASSET_METADATA_AUDIO_SAMPLE_RATE),
		fmt.Sprintf("%s AS %s", ASSET_METADATA_AUDIO_TABLE_BIT_RATE, ASSET_METADATA_AUDIO_BIT_RATE),
	}
}
