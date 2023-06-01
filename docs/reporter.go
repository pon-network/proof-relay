package docs

import databaseTypes "github.com/bsn-eng/pon-golang-types/database"

// swagger:route POST /blocksubmissions Reporter-API builderSubmission
// Get Block Submissions Of Builders.
// responses:
//   200: validResponse
//   204: emptyResponse
//	 400: invalidParameter
//   500: serverError

// Blocks Provided Correctly
// swagger:response validResponse
type Response struct {
	// Builder Bids
	Body []databaseTypes.BuilderBlockDatabase `json:"builder_bids"`
}

// No Builder Submissions
// swagger:response emptyResponse
type EmptyResponse struct {
	// Empty List Of Bids
	Body []databaseTypes.BuilderBlockDatabase `json:"slot"`
}

// Invalid Parameter Provided
// swagger:response invalidParameter
type invalidParameter struct {
	// Error Parameters
	Error string `json:"error"`
}

// Server Error
// swagger:response serverError
type serverError struct {
	// Error In The Server
	Error string `json:"error"`
}

// swagger:parameters builderSubmission
type ReporterSlot struct {
	// Slot Number From Which Needed
	SlotLower uint64 `json:"slot_lower"`
	// Slot Number To Which Needed
	SlotUpper uint64 `json:"slot_upper"`
}

// swagger:route POST /payloaddelivered Reporter-API proposerPayloadDelivered
// Get Proposer Payload Delivered.
// responses:
//   200: validResponsePayloaddelivered
//   204: emptyResponse
//	 400: invalidParameter
//   500: serverError

// Blocks Provided Correctly
// swagger:response validResponsePayloaddelivered
type payloadDelivered struct {
	// Delivered Payloads
	Body []databaseTypes.ValidatorDeliveredPayloadDatabase `json:"payload_delivered"`
}

// swagger:parameters proposerPayloadDelivered
type reporterSlotPayloadDelivered struct {
	// Slot Number From Which Needed
	SlotLower uint64 `json:"slot_lower"`
	// Slot Number To Which Needed
	SlotUpper uint64 `json:"slot_upper"`
}

// swagger:route POST /proposerblindedblocks Reporter-API proposerBlindedBlock
// Get Proposer Payload Delivered.
// responses:
//   200: validResponseProposerblindedblocks
//   204: emptyResponse
//	 400: invalidParameter
//   500: serverError

// swagger:parameters proposerBlindedBlock
type reporterSlotProposerBlindedBlock struct {
	// Slot Number From Which Needed
	SlotLower uint64 `json:"slot_lower"`
	// Slot Number To Which Needed
	SlotUpper uint64 `json:"slot_upper"`
}

// Blocks Provided Correctly
// swagger:response validResponseProposerblindedblocks
type proposerBlindedblocks struct {
	// Blinded Beacon Blocks
	Body []databaseTypes.ValidatorReturnedBlockDatabase `json:"builder_blinded_blocks" description:"ss"`
}
