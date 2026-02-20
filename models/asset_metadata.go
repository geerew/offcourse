package models

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// AssetMetadata defines metadata for a video asset
//
// This is not a database table. Instead it is a combination of an assets
// metadata
type AssetMetadata struct {
	AssetID string

	// Joins
	VideoMetadata *VideoMetadata
	AudioMetadata *AudioMetadata
}
