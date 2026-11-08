// Age-group cohorts — stable wire values sent as the `ageGroup` field and
// stored in app_user.age_group on the server.
//
// Keep in sync with server/obj/age_group.go.
export const AgeGroup = {
  U13: "u13",   // 13-17, no parental consent on file (strictest; fallback when unknown)
  U13P: "u13p", // 13-17, with parental consent on file
  U18: "u18",   // 18+
} as const;

export type AgeGroup = (typeof AgeGroup)[keyof typeof AgeGroup];

// UI-only marker for the under-13 registration block. Not stored and never
// sent to the server — selecting it disables the submit button.
export const AGE_GROUP_UNDER_13 = "under13";
