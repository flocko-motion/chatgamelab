/* eslint-disable */
/* tslint:disable */
// @ts-nocheck
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

export type WorkshopSort =
  | "name"
  | "createdAt"
  | "modifiedAt"
  | "participantCount";

export type WorkshopFilter = "active";

export type VoiceChoice = "male" | "female" | "any" | "custom";

export type VerifiedPageSort = "status" | "organisation" | "createdAt";

export type VerifiedPageFilter = "waiting" | "approved";

export type UserSort = "name" | "email" | "role" | "organisation" | "createdAt";

export type UserFilter = "noGuests";

export type Toggle = "images" | "audio-out" | "audio-in" | "audio-live";

export type Tier = "low" | "medium" | "high";

export type Subgenre =
  | "adventure-rounds"
  | "npc-live"
  | "npc-rounds"
  | "discussion-live";

export type SortDir = "asc" | "desc";

export type Role = "admin" | "head" | "staff" | "guest" | "individual";

export type Platform = "mock" | "openai" | "mistral";

export type ParticipantSort = "name" | "role" | "gamesCount" | "createdAt";

export type OwnApiKeySort = "createdAt" | "name" | "platform" | "status";

export type OverReason = "content_policy";

export type OrganisationSort = "name" | "createdAt" | "registeredAt";

export type OrganisationKeySort = "name" | "owner" | "platform";

export type OrganisationFilter = "recentlyRegistered";

export type NotificationType =
  | "newOrganisations"
  | "pagesWaiting"
  | "pageApproved"
  | "pageTurnedDown";

export type ModelType =
  | "tooling"
  | "story"
  | "image"
  | "live"
  | "speech"
  | "hearing";

export type Mode = "live" | "rounds";

export type MemberSort = "name" | "email" | "role";

export type Medium = "text" | "audio" | "image";

export type Level = "info" | "warning" | "error";

export type InviteStatus =
  | "pending"
  | "accepted"
  | "declined"
  | "expired"
  | "revoked";

export type InviteSort = "role" | "organisation" | "status" | "modifiedAt";

export type Genre = "adventure" | "npc" | "discussion";

export type GameSort =
  | "createdAt"
  | "modifiedAt"
  | "name"
  | "playCount"
  | "visibility"
  | "creator";

export type GameFilter =
  | "own"
  | "others"
  | "organisation"
  | "public"
  | "favorites"
  | "sponsored";

export type GameField =
  | "scenario"
  | "gameStart"
  | "imageStyle"
  | "voiceStyle"
  | "statusFields";

export type EventType =
  | "workshop_updated"
  | "game_created"
  | "game_updated"
  | "game_deleted"
  | "members_updated"
  | "connected";

export type EngineModelsReset = "assignments" | "entries";

export interface AcceptInviteResp {
  authToken?: string;
  message: string;
  user?: NewGuestView;
}

export interface AcknowledgeNotificationReq {
  /** @format date-time */
  at?: string;
  /** @format uuid */
  subject?: string;
  type: NotificationType;
}

export interface ApiKey {
  /** @format uuid */
  id: string;
  isDefault: boolean;
  keyShortened: string;
  lastErrorCode?: string;
  lastUsageSuccess?: boolean;
  meta: Meta;
  name: string;
  platform: string;
  /** @format uuid */
  userId: string;
  userName: string;
}

export interface ApiKeyInfoResp {
  linkedShares: KeyShareView[];
  share?: KeyShareView;
}

export interface ApiKeyShare {
  apiKey?: ApiKey;
  /** @format uuid */
  apiKeyId: string;
  game?: Game;
  /** @format uuid */
  gameShareId?: string;
  /** @format uuid */
  id: string;
  institution?: Institution;
  isPrivateShare?: boolean;
  isUserDefault: boolean;
  meta: Meta;
  remaining?: number;
  user?: User;
  workshop?: Workshop;
}

export interface ApiKeyView {
  /** @format uuid */
  id: string;
  isDefault: boolean;
  keyShortened: string;
  lastErrorCode?: string;
  lastUsageSuccess?: boolean;
  meta: KeyMeta;
  name: string;
  platform: string;
  /** @format uuid */
  userId: string;
  userName: string;
}

export interface ApproveVerifiedPageReq {
  approved: boolean;
}

export interface AssignEngineModelReq {
  model: string;
  platform: Platform;
  quality?: string;
  tier: Tier;
  type: ModelType;
}

export interface AvailableKey {
  isDefault: boolean;
  name: string;
  platform: string;
  /** @format uuid */
  shareId: string;
  source: string;
}

export interface BeatRequest {
  ms: number;
}

export interface BlockFrequency {
  block: string;
  frequency: number;
}

export interface BlockPrompt {
  default: string;
  edited: boolean;
  name: string;
  text: string;
}

export interface Branding {
  aiInsightsUrl: string;
  apiKeyFaqUrl: string;
  contactUrl: string;
  creditName: string;
  creditUrl: string;
  educatorInfoUrl: string;
  gameTipsUrl: string;
  helpUrl: string;
  operatorName: string;
  operatorUrl: string;
  privacyUrl: string;
  reportGameUrl: string;
  siteName: string;
  supportEmail: string;
  termsUrl: string;
  websiteUrl: string;
}

export interface CheckNameResp {
  available: boolean;
  profane?: boolean;
}

export interface Choice {
  tier: Tier;
  toggles: Toggles;
}

export interface ChooseRequest {
  choice?: Choice;
  offer: string;
}

export interface CompleteRegistrationReq {
  acceptTerms?: boolean;
  ageGroup: string;
  email?: string;
  language?: string;
  name: string;
  organisation?: OrganisationSignUpReq;
}

export interface ConnectRequest {
  sdp: string;
  seat?: string;
}

export interface ConnectResponse {
  sdp: string;
}

export interface ConsentStatusView {
  /** @format date-time */
  consentedAt?: string;
  consentedVersion?: string;
  currentVersion: string;
  pending: boolean;
  /** @format date-time */
  withdrawnAt?: string;
}

export interface ConsentToDeclarationReq {
  acceptDeclaration: boolean;
  city: string;
  declarationVersion: string;
  email: string;
  operator: string;
  personFunction: string;
  personName: string;
  postalCode: string;
  promptConstraint: string;
  street: string;
}

export interface ConstraintFrequencyRequest {
  block: string;
  frequency: number;
}

export interface CreateApiKeyReq {
  key: string;
  name: string;
  platform: string;
}

export interface CreateGameReq {
  css?: string;
  description?: string;
  /** @format byte */
  firstImage?: Blob;
  firstMessage?: string;
  firstStatus?: string;
  genre?: string;
  /** @format byte */
  icon?: Blob;
  imageStyle?: string;
  name?: string;
  public?: boolean;
  statusFields?: string;
  systemMessageGameStart?: string;
  systemMessageScenario?: string;
  theme?: string;
  voiceStyle?: string;
  voices?: Record<string, string>;
}

export interface CreateGameShareReq {
  aiQualityTier?: string;
  maxSessions?: number;
  /** @format uuid */
  sponsorKeyShareId?: string;
  /** @format uuid */
  workshopId?: string;
}

export interface CreateInstitutionInviteReq {
  institutionId: string;
  invitedEmail?: string;
  invitedUserId?: string;
  role: string;
}

export interface CreateInstitutionReq {
  city?: string;
  name: string;
  operator?: string;
  postalCode?: string;
  street?: string;
}

export interface CreateRequest {
  game?: string;
  language?: string;
  mode?: Mode;
  once: string;
  player?: string;
  promptOverride?: Record<string, string>;
  request?: string;
  share?: string;
}

export interface CreateResponse {
  id: string;
}

export interface CreateUserReq {
  email?: string;
  name: string;
}

export interface CreateVerifiedPageReq {
  callbackUrl: string;
  pageUrl: string;
}

export interface CreateWorkshopEmailInviteReq {
  email: string;
  workshopId: string;
}

export interface CreateWorkshopInviteReq {
  expiresAt?: string;
  maxUses?: number;
  workshopId: string;
}

export interface CreateWorkshopReq {
  active: boolean;
  institutionId?: string;
  name: string;
  public: boolean;
}

export interface DeclarationResp {
  bindingLanguage: string;
  declaration: DeclarationView;
  defaultPromptConstraint?: string;
  maxPromptConstraintLength: number;
  youthProtectionSuggestions: YouthProtectionSuggestion[];
}

export interface DeclarationView {
  consent: string;
  date: string;
  hash: string;
  text: string;
  version: string;
}

export interface DevLoginReq {
  role: string;
}

export interface DevLoginResp {
  token: string;
  /** @format uuid */
  userId: string;
}

export interface EdgeDetail {
  from: string;
  kind: string;
  recent: Sample[];
  to: string;
}

export interface EngineModelsPlatform {
  assignments: ModelSlotView[];
  configurable: boolean;
  id: Platform;
  models: ModelTableRowView[];
  name: string;
}

export interface EngineModelsResp {
  platforms: EngineModelsPlatform[];
}

export interface Error {
  /** Machine-readable code: an app code, or the default code of the status (bad_request, unauthorized, forbidden, not_found, conflict, gone, precondition_failed, too_large, unprocessable, too_many_requests, not_implemented, bad_gateway, unavailable, gateway_timeout, internal, method_not_allowed, not_acceptable, unsupported_media_type). */
  code: string;
  /** The message. On 5xx always the status text. */
  error: string;
  /** On 5xx only: the id the full error is logged under. */
  id?: string;
}

export interface FavouriteResp {
  favourite: boolean;
}

export interface FlowchartResponse {
  flowchart: string;
}

export interface Game {
  cloneCount: number;
  /** @format uuid */
  creatorId?: string;
  creatorName?: string;
  css: string;
  description: string;
  /** @format byte */
  firstImage: Blob;
  firstMessage?: string;
  firstStatus?: string;
  genre: string;
  /** @format byte */
  icon: Blob;
  /** @format uuid */
  id: string;
  imageStyle: string;
  meta: Meta;
  name: string;
  /** @format uuid */
  originalCreatorId?: string;
  originalCreatorName?: string;
  /** @format uuid */
  originallyCreatedBy?: string;
  playCount: number;
  public: boolean;
  /** @format uuid */
  publicSponsoredApiKeyShareId?: string;
  statusFields: string;
  systemMessageGameStart: string;
  systemMessageScenario: string;
  tags: GameTag[];
  theme?: string;
  voiceStyle: string;
  voices?: Record<string, string>;
  /** @format uuid */
  workshopId?: string;
}

export interface GameApiKeyStatusResp {
  available: boolean;
}

export interface GameEmbedResp {
  embed?: GameEmbedView;
}

export interface GameEmbedView {
  enabled: boolean;
  /** @format uuid */
  id: string;
  keyName: string;
  mode: string;
  platform: string;
  /** @format uuid */
  sponsorKeyShareId?: string;
  token: string;
  verifiedPageApproved: boolean;
  /** @format uuid */
  verifiedPageId?: string;
  verifiedPageUrl?: string;
}

export interface GameFile {
  css: string;
  description: string;
  genre: string;
  /** @format uuid */
  id: string;
  image_style: string;
  name: string;
  status_fields: string;
  system_message_game_start: string;
  system_message_scenario: string;
  theme?: string;
  voice_style?: string;
  voices?: Record<string, string>;
}

export interface GameMeta {
  /** @format date-time */
  createdAt: string;
  /** @format uuid */
  createdBy?: string;
  /** @format date-time */
  modifiedAt: string;
}

export interface GameSponsorReq {
  /** @format uuid */
  shareId: string;
}

export interface GameTag {
  /** @format uuid */
  gameId: string;
  /** @format uuid */
  id: string;
  meta: Meta;
  tag: string;
}

export interface GameTemplate {
  css?: string;
  description: string;
  /** @format byte */
  firstImage?: Blob;
  firstMessage?: string;
  firstStatus?: string;
  genre: string;
  hasContent: boolean;
  /** @format byte */
  icon: Blob;
  imageStyle?: string;
  mayEditCard: boolean;
  mayEditContent: boolean;
  mayPublish: boolean;
  name: string;
  public: boolean;
  statusFields?: string;
  systemMessageGameStart?: string;
  systemMessageScenario?: string;
  theme?: string;
  voiceStyle?: string;
  voices?: Record<string, string>;
}

export interface GameView {
  cloneCount: number;
  /** @format date-time */
  coverAt?: string;
  /** @format uuid */
  creatorId?: string;
  creatorName?: string;
  css?: string;
  description: string;
  /** @format byte */
  firstImage?: Blob;
  firstMessage?: string;
  firstStatus?: string;
  genre: string;
  hasContent: boolean;
  hasSettings: boolean;
  /** @format byte */
  icon: Blob;
  /** @format uuid */
  id: string;
  imageStyle?: string;
  isFavorite: boolean;
  mayCopy: boolean;
  mayDelete: boolean;
  mayEditCard: boolean;
  mayEditContent: boolean;
  mayEditSettings: boolean;
  mayPresetPrompts: boolean;
  mayPublish: boolean;
  mayShare: boolean;
  meta: GameMeta;
  name: string;
  /** @format uuid */
  organisationId?: string;
  /** @format uuid */
  originallyCreatedBy?: string;
  playCount: number;
  playability: string;
  public: boolean;
  /** @format uuid */
  publicSponsoredApiKeyShareId?: string;
  statusFields?: string;
  systemMessageGameStart?: string;
  systemMessageScenario?: string;
  theme?: string;
  voiceStyle?: string;
  voices?: Record<string, string>;
  /** @format uuid */
  workshopId?: string;
}

export interface GameViewPage {
  items: GameView[];
  total: number;
}

export interface GenreInfo {
  fields: GameField[];
  id: Genre;
  needs: Toggle[];
  subgenres: SubgenreInfo[];
  voiceSlots: VoiceSlot[];
}

export interface GenrePrompt {
  name: string;
  text: string;
}

export interface GetCurrentUserResp {
  institutionSuspension?: string;
  mayConsentForInstitution: boolean;
  maySuspendInstitutions: boolean;
  state: string;
  user?: UserView;
}

export interface GetVersionResp {
  version: string;
}

export interface GraphResponse {
  graph: string;
}

export interface GuardrailEditRequest {
  text: string;
}

export interface GuardrailReport {
  edited: boolean;
  frequencies: BlockFrequency[];
  mayEdit: boolean;
  readBy: string[];
  resolved: string;
  source?: GuardrailSource;
  text: string;
}

export interface GuardrailSource {
  level: string;
  name?: string;
}

export interface GuestGameInfoResp {
  description?: string;
  /** @format uuid */
  gameId: string;
  name: string;
  playability: string;
  public: boolean;
  remaining?: number;
}

export interface GuestLoginReq {
  token: string;
}

export interface GuestLoginResp {
  token: string;
  /** @format uuid */
  userId: string;
}

export interface GuestTokenResp {
  token: string;
}

export interface InjectionRequest {
  kind: string;
  text: string;
}

export interface InputRequest {
  /** @format byte */
  audio?: Blob;
  text?: string;
}

export interface Institution {
  /** @format uuid */
  freeUseApiKeyShareId?: string;
  /** @format uuid */
  id: string;
  meta: Meta;
  name: string;
  promptConstraints?: string;
}

export interface InviteCardView {
  /** @format date-time */
  acceptedAt?: string;
  /** @format uuid */
  acceptedBy?: string;
  /** @format date-time */
  expiresAt?: string;
  /** @format uuid */
  id: string;
  /** @format uuid */
  institutionId: string;
  institutionName: string;
  inviteToken?: string;
  invitedEmail?: string;
  /** @format uuid */
  invitedUserId?: string;
  maxUses?: number;
  mayAnswer: boolean;
  mayRevoke: boolean;
  meta: InviteMeta;
  role: string;
  status: string;
  usesCount: number;
  /** @format uuid */
  workshopId?: string;
  workshopName?: string;
}

export interface InviteCardViewPage {
  items: InviteCardView[];
  total: number;
}

export interface InviteMeta {
  /** @format date-time */
  createdAt: string;
  /** @format date-time */
  modifiedAt: string;
}

export interface InviteView {
  /** @format date-time */
  acceptedAt?: string;
  /** @format uuid */
  acceptedBy?: string;
  /** @format date-time */
  expiresAt?: string;
  /** @format uuid */
  id: string;
  /** @format uuid */
  institutionId: string;
  inviteToken?: string;
  invitedEmail?: string;
  /** @format uuid */
  invitedUserId?: string;
  maxUses?: number;
  meta: InviteMeta;
  role: string;
  status: string;
  usesCount: number;
  /** @format uuid */
  workshopId?: string;
}

export interface JournalEntry {
  detail?: string;
  kind: string;
  level: Level;
  source: string;
  /** @format date-time */
  time: string;
}

export interface KeyMeta {
  /** @format date-time */
  createdAt: string;
  /** @format date-time */
  modifiedAt: string;
}

export interface KeyShareMeta {
  /** @format date-time */
  createdAt: string;
  /** @format uuid */
  createdBy?: string;
}

export interface KeyShareView {
  apiKey: ApiKeyView;
  /** @format uuid */
  apiKeyId: string;
  game?: NamedRef;
  /** @format uuid */
  gameShareId?: string;
  /** @format uuid */
  id: string;
  institution?: NamedRef;
  isPrivateShare?: boolean;
  mayRevoke: boolean;
  mayShare: boolean;
  meta: KeyShareMeta;
  remaining?: number;
  self: boolean;
  /** @format uuid */
  sourceShareId?: string;
}

export interface KeyShareViewPage {
  items: KeyShareView[];
  total: number;
}

export interface Language {
  iso: string;
  label: string;
}

export interface MakeGameCoverReq {
  hint: string;
}

export interface Message {
  body: string;
  /** @format date-time */
  sentAt: string;
  subject: string;
  to: string;
}

export interface MessageResp {
  message: string;
}

export interface Meta {
  /** @format date-time */
  createdAt?: string;
  /** @format uuid */
  createdBy: string | null;
  /** @format date-time */
  modifiedAt?: string;
  /** @format uuid */
  modifiedBy: string | null;
}

export interface Model {
  name: string;
  params?: Record<string, string>;
}

export interface ModelFacts {
  in: Medium[];
  out: Medium[];
  price: Price;
  read: string;
  source: string;
  window?: number;
  windowRead?: string;
  windowSource?: string;
}

export interface ModelFactsOverride {
  in?: Medium[];
  out?: Medium[];
  price: Price;
  since: string;
  window?: number;
}

export interface ModelSlotView {
  assigned: boolean;
  default: string;
  defaultQuality?: string;
  model: string;
  /** @format date-time */
  modifiedAt?: string;
  quality?: string;
  tier: Tier;
  type: ModelType;
}

export interface ModelTableRowView {
  default?: ModelFacts;
  facts: ModelFacts;
  inUse: boolean;
  model: string;
  /** @format date-time */
  modifiedAt?: string;
  overridden: boolean;
  stale: boolean;
}

export interface NamedRef {
  /** @format uuid */
  id: string;
  name: string;
}

export interface NewGuestView {
  /** @format uuid */
  id: string;
  language: string;
  name: string;
}

export interface NodeDetail {
  injectable?: boolean;
  injections?: string[];
  inputs: Sample[];
  name: string;
  observer?: ObserverState;
  outputs: Sample[];
  phase: string;
  prompts?: BlockPrompt[];
  role: string;
  thread?: ThreadFill;
  transcript?: TranscriptLine[];
  type: string;
  usage?: UsageRecord;
}

export interface NotificationView {
  /** @format date-time */
  at: string;
  count: number;
  /** @format uuid */
  organisationId?: string;
  /** @format uuid */
  subject?: string;
  type: NotificationType;
}

export interface NotificationsResp {
  notifications: NotificationView[];
}

export interface ObserverEntry {
  /** @format date-time */
  at: string;
  kind: string;
  late?: boolean;
  speaker?: string;
  text: string;
}

export interface ObserverFigures {
  backgrounds: number;
  calls: number;
  directions: number;
  dropped: number;
  errors: number;
  flushes: number;
  fragments: number;
  injected: number;
  largestLateMs: number;
  lastCallMs: number;
  late: number;
  lines: number;
  quiet: number;
  reminders: number;
  silenceNotices: number;
}

export interface ObserverState {
  absentMs: number;
  awaiting: boolean;
  beatMs: number;
  characterAgoMs: number;
  figures: ObserverFigures;
  gapMs: number;
  held?: string;
  inARow: number;
  inARowLimit: number;
  log: ObserverEntry[];
  pending: number;
  playerAgoMs: number;
  pool: number;
  quietForMs: number;
  quietMs: number;
  standingDown: boolean;
  thinking: boolean;
  thinkingForMs: number;
}

export interface OfferOption {
  customizable: boolean;
  default: Choice;
  name: string;
  offer: string;
  platform: Platform;
  tiers: TierOption[];
}

export interface Options {
  choosing: boolean;
  offers: OfferOption[];
  settled?: Settled;
}

export interface OrganisationConsentView {
  city?: string;
  declaration: ConsentStatusView;
  /** @format uuid */
  id: string;
  name: string;
  operator?: string;
  postalCode?: string;
  promptConstraints?: string;
  street?: string;
}

export interface OrganisationMeta {
  /** @format date-time */
  createdAt: string;
  /** @format date-time */
  modifiedAt: string;
}

export interface OrganisationRegistrationView {
  /** @format date-time */
  at: string;
  email: string;
  personFunction: string;
  personName: string;
}

export interface OrganisationSignUpReq {
  acceptDeclaration: boolean;
  city: string;
  declarationVersion: string;
  institutionName: string;
  operator: string;
  personFunction: string;
  personName: string;
  postalCode: string;
  promptConstraint: string;
  street: string;
}

export interface OrganisationSuspensionView {
  /** @format uuid */
  id: string;
  /** @format date-time */
  suspendedAt?: string;
  suspendedReason?: string;
}

export interface OrganisationView {
  city?: string;
  declaration?: ConsentStatusView;
  freeUseAiQualityTier?: string;
  /** @format uuid */
  freeUseApiKeyShareId?: string;
  hasDetails: boolean;
  hasRecord: boolean;
  hasSettings: boolean;
  /** @format uuid */
  id: string;
  mayCreateWorkshop: boolean;
  mayDelete: boolean;
  mayEditCard: boolean;
  mayEditDetails: boolean;
  mayEditSettings: boolean;
  mayInviteHead: boolean;
  mayInviteStaff: boolean;
  mayListApiKeys: boolean;
  maySuspend: boolean;
  members: UserView[];
  meta: OrganisationMeta;
  name: string;
  operator?: string;
  postalCode?: string;
  promptConstraints?: string;
  registration?: OrganisationRegistrationView;
  street?: string;
  suspended: boolean;
  /** @format date-time */
  suspendedAt?: string;
  suspendedReason?: string;
}

export interface OrganisationViewPage {
  items: OrganisationView[];
  total: number;
}

export interface Over {
  /** @format date-time */
  at: string;
  reason: OverReason;
}

export interface OwnApiKeyView {
  key: ApiKeyView;
  selfShare: KeyShareView;
  shares: KeyShareView[];
}

export interface OwnApiKeyViewPage {
  items: OwnApiKeyView[];
  total: number;
}

export interface ParticipantMeta {
  /** @format date-time */
  createdAt: string;
}

export interface Place {
  x: number;
  y: number;
}

export interface PlatformInfo {
  configurable: boolean;
  genres: Genre[];
  id: Platform;
  models: Record<string, ModelFacts>;
  name: string;
  tiers: TierModels[];
  toggles: Toggles;
  userKeys: boolean;
}

export interface Preference {
  offer?: string;
  platform: Platform;
  tier: Tier;
  toggles: Toggles;
}

export interface Price {
  audioInputPerMTok?: number;
  audioOutputPerMTok?: number;
  audioPerMinute?: number;
  cachedAudioInputPerMTok?: number;
  cachedInputPerMTok?: number;
  inputPerMTok?: number;
  outputPerMTok?: number;
  perImage?: number;
}

export interface PrivateShareStatusResp {
  shares: ShareLinkView[];
}

export interface ProbeEngineModelReq {
  model: string;
  platform: Platform;
  type: ModelType;
}

export interface ProbeEngineModelResp {
  answers: boolean;
  cost: number;
  message?: string;
  priced: boolean;
  problem?: string;
}

export interface PromptEditRequest {
  text: string;
}

export interface PublicWorkshopGame {
  description: string;
  /** @format uuid */
  id: string;
  name: string;
  play?: PublicWorkshopPlay;
}

export interface PublicWorkshopLink {
  description?: string;
  title: string;
  url: string;
}

export interface PublicWorkshopPage {
  description: string;
  games: PublicWorkshopGame[];
  links?: PublicWorkshopLink[];
  name: string;
  playAvailable: boolean;
}

export interface PublicWorkshopPlay {
  limit: number;
  remaining: number;
  token: string;
}

export interface ResetEngineModelsReq {
  platform: Platform;
  what: EngineModelsReset;
}

export interface ReviewGameReq {
  description: string;
  genre: string;
  imageStyle: string;
  name: string;
  statusFields: string;
  systemMessageGameStart: string;
  systemMessageScenario: string;
  voiceStyle: string;
}

export interface ReviewGameResp {
  field: string;
  missing: boolean;
  text: string;
}

export interface RolesResp {
  roles: Role[];
}

export interface Sample {
  /** @format date-time */
  at: string;
  kind: string;
  peer: string;
  value: string;
}

export interface SchemafAuth0LoginRequest {
  id_token: string;
}

export interface SchemafErrorDetails {
  code: string;
  /** The full error. */
  error: string;
  id: string;
  method: string;
  /** The misuse of the error API, if any. */
  misuse?: string;
  path: string;
  status: number;
  /** The caller, if authenticated. */
  subject?: string;
  /** @format date-time */
  time: string;
}

export interface SchemafLoginResponse {
  subject: string;
  token: string;
}

export interface SchemafRefreshResponse {
  token: string;
}

export interface SchemafUpdateRequest {
  version: string;
}

export interface SchemafUpdateState {
  available: {
    notes: string;
    version: string;
  }[];
  check?: {
    finished: string;
    message: string;
    started: string;
    state: "checking" | "checked" | "failed" | "rejected";
    trigger: "timer" | "app" | "operator";
  };
  checked?: string;
  install?: {
    finished: string;
    from: string;
    message: string;
    started: string;
    state:
      | "installing"
      | "installed"
      | "rolled-back"
      | "failed"
      | "rollback-failed";
    version: string;
  };
  request?: {
    message: string;
    state: "accepted" | "rejected";
    time: string;
    version: string;
  };
  /** The running version; empty in development. */
  running: string;
  status: "updater not enabled" | "idle" | "installing" | "checking";
}

export interface SchemafUser {
  id: string;
}

export interface SessionDeletedResp {
  status: string;
}

export interface SessionEvent {
  Stream: string;
  Value: string;
}

export interface SessionView {
  /** @format uuid */
  apiKeyShareId?: string;
  /** @format date-time */
  createdAt: string;
  /** @format uuid */
  gameId: string;
  gameName?: string;
  /** @format uuid */
  gameShareId?: string;
  /** @format uuid */
  id: string;
  language: string;
  messagesSent: number;
  /** @format date-time */
  modifiedAt: string;
  platform: string;
  /** @format uuid */
  userId: string;
  /** @format uuid */
  workshopId?: string;
}

export interface SetActiveWorkshopReq {
  /** @format uuid */
  workshopId?: string;
}

export interface SetEngineModelEntryReq {
  facts?: ModelFactsOverride;
  model: string;
  platform: Platform;
}

export interface SetGameEmbedReq {
  enabled: boolean;
  mode: string;
  /** @format uuid */
  sponsorKeyShareId?: string;
  /** @format uuid */
  verifiedPageId?: string;
}

export interface SetInstitutionFreeUseKeyReq {
  /** @format uuid */
  shareId?: string;
}

export interface SetSystemFreeUseApiKeyReq {
  /** @format uuid */
  apiKeyId?: string;
}

export interface SetUserRoleReq {
  /** @format uuid */
  institutionId?: string;
  role: string;
  /** @format uuid */
  workshopId?: string;
}

export interface SetWorkshopApiKeyReq {
  apiKeyShareId?: string;
}

export interface Settled {
  changes?: string[];
  choice: Choice;
  offer: string;
  platform: Platform;
  preference: Preference;
}

export interface ShareApiKeyReq {
  /** @format uuid */
  institutionId?: string;
  /** @format uuid */
  userId?: string;
  /** @format uuid */
  workshopId?: string;
}

export interface ShareLinkView {
  aiQualityTier?: string;
  /** @format uuid */
  apiKeyShareId: string;
  /** @format date-time */
  createdAt: string;
  /** @format uuid */
  createdBy?: string;
  embed: boolean;
  enabled: boolean;
  /** @format uuid */
  gameId: string;
  gameName?: string;
  /** @format uuid */
  id: string;
  /** @format uuid */
  institutionId?: string;
  keyName: string;
  mayManage: boolean;
  remaining?: number;
  shareUrl: string;
  source: string;
  token: string;
  /** @format uuid */
  workshopId?: string;
  workshopName?: string;
}

export interface SiteResp {
  branding: Branding;
  defaults: Branding;
}

export interface SiteRuleTexts {
  u13: string;
  u13p: string;
  u18: string;
}

export interface Snapshot {
  choosing: boolean;
  conversation?: string;
  over?: Over;
  phases: Record<string, string>;
  props: Record<string, string>;
  settled?: Settled;
  started: boolean;
  usage: UsageReport;
}

export interface SubgenreInfo {
  id: Subgenre;
  mode: Mode;
  needs: Toggle[];
  prompts: GenrePrompt[];
  voiceSlots: VoiceSlot[];
}

export interface SuspendOrganisationReq {
  reason?: string;
}

export interface SystemConstraintsResp {
  promptConstraintU13?: string;
  promptConstraintU13p?: string;
  promptConstraintU18?: string;
}

export interface SystemSettings {
  /** @format date-time */
  createdAt?: string;
  defaultAiQualityTier: string;
  freeUseAiQualityTier?: string;
  /** @format uuid */
  freeUseApiKeyId?: string;
  freeUseApiKeyName?: string;
  freeUseApiKeyPlatform?: string;
  freeUseApiKeyWorking?: boolean;
  /** @format uuid */
  id: string;
  /** @format date-time */
  modifiedAt?: string;
  promptConstraintDefaults: SiteRuleTexts;
  promptConstraintU13?: string;
  promptConstraintU13p?: string;
  promptConstraintU18?: string;
  sessionRetentionHours: number;
}

export interface ThemeInfo {
  id: string;
}

export interface ThreadFill {
  tokens: number;
  window: number;
}

export interface TierModels {
  id: Tier;
  label: string;
  models: Record<string, Model>;
}

export interface TierOption {
  label: string;
  models: Record<string, string>;
  tier: Tier;
  toggles: Toggles;
}

export interface ToggleRequest {
  on: boolean;
  toggle: Toggle;
}

export interface Toggles {
  "audio-in": boolean;
  "audio-live": boolean;
  "audio-out": boolean;
  images: boolean;
}

export interface Topology {
  dev?: boolean;
  edges: TopologyEdge[];
  imagery: string;
  inputs: string[];
  language?: string;
  name: string;
  nodes: TopologyNode[];
  seats?: string[];
  switches: string[];
  theme?: string;
  title?: string;
}

export interface TopologyEdge {
  from: string;
  kind: string;
  port?: string;
  to: string;
}

export interface TopologyNode {
  field?: string;
  name: string;
  place?: Place;
  role: string;
}

export interface TranscriptLine {
  speaker: string;
  text: string;
}

export interface UpdateApiKeyReq {
  name: string;
}

export interface UpdateGameReq {
  css?: string;
  description?: string;
  /** @format byte */
  firstImage?: Blob;
  firstMessage?: string;
  firstStatus?: string;
  genre?: string;
  /** @format byte */
  icon?: Blob;
  imageStyle?: string;
  name?: string;
  public?: boolean;
  statusFields?: string;
  systemMessageGameStart?: string;
  systemMessageScenario?: string;
  theme?: string;
  voiceStyle?: string;
  voices?: Record<string, string>;
}

export interface UpdateGameShareReq {
  aiQualityTier?: string;
  maxSessions?: number;
}

export interface UpdateGameYAMLReq {
  css: string;
  description: string;
  genre: string;
  /** @format uuid */
  id: string;
  image_style: string;
  name: string;
  status_fields: string;
  system_message_game_start: string;
  system_message_scenario: string;
  theme?: string;
  voice_style?: string;
  voices?: Record<string, string>;
}

export interface UpdateInstitutionReq {
  name?: string;
  promptConstraints?: string;
}

export interface UpdateMyLanguageReq {
  language: string;
}

export interface UpdateSiteBrandingReq {
  branding: Record<string, string>;
}

export interface UpdateSystemSettingsReq {
  defaultAiQualityTier?: string;
  freeUseAiQualityTier?: string;
  promptConstraintU13?: string;
  promptConstraintU13p?: string;
  promptConstraintU18?: string;
  sessionRetentionHours?: number;
}

export interface UpdateUserReq {
  ageGroup?: string;
  aiQualityTier?: string;
  /** @format uuid */
  defaultApiKeyShareId?: string;
  email?: string;
  language?: string;
  name?: string;
}

export interface UpdateVerifiedPageReq {
  callbackUrl: string;
  pageUrl: string;
}

export interface UpdateWorkshopReq {
  active?: boolean;
  aiQualityTier?: string;
  allowGameSharing?: boolean;
  isPaused?: boolean;
  name?: string;
  promptConstraints?: string;
  public?: boolean;
  publicDescription?: string;
  publicLinks?: PublicWorkshopLink[];
  publicSlug?: string;
  showOtherParticipantsGames?: boolean;
  showPublicGames?: boolean;
}

export interface UsageRecord {
  audioInputTokens?: number;
  audioOutputTokens?: number;
  audioSeconds?: number;
  cachedAudioInputTokens?: number;
  cachedInputTokens?: number;
  cost: number;
  images?: number;
  inputTokens?: number;
  model: string;
  node?: string;
  outputTokens?: number;
  priced: boolean;
}

export interface UsageReport {
  byModel: UsageRecord[];
  byNode: UsageRecord[];
  complete: boolean;
  totalCost: number;
}

export interface User {
  ageGroup?: string;
  aiQualityTier?: string;
  apiKeys: ApiKeyShare[];
  auth0Id?: string;
  /** @format date-time */
  deletedAt?: string;
  email?: string;
  /** @format uuid */
  id: string;
  language: string;
  meta: Meta;
  name: string;
  role?: UserRole;
}

export interface UserMeta {
  /** @format date-time */
  createdAt: string;
  /** @format date-time */
  modifiedAt: string;
}

export interface UserRole {
  /** @format uuid */
  id: string;
  institution?: Institution;
  meta: Meta;
  role: Role;
  /** @format uuid */
  userId: string;
  workshop?: Workshop;
}

export interface UserRoleInvite {
  /** @format date-time */
  acceptedAt?: string;
  /** @format uuid */
  acceptedBy?: string;
  /** @format date-time */
  expiresAt?: string;
  /** @format uuid */
  id: string;
  /** @format uuid */
  institutionId: string;
  inviteToken?: string;
  invitedEmail?: string;
  /** @format uuid */
  invitedUserId?: string;
  maxUses?: number;
  meta: Meta;
  role: Role;
  status: InviteStatus;
  usesCount: number;
  /** @format uuid */
  workshopId?: string;
}

export interface UserRoleRef {
  institution?: NamedRef;
  role: string;
  workshop?: NamedRef;
}

export interface UserSecretsView {
  token?: string;
  words?: string;
}

export interface UserStatsView {
  gamesCreated: number;
  gamesPlayed: number;
  messagesSent: number;
  totalPlaysOnGames: number;
}

export interface UserView {
  ageGroup?: string;
  aiQualityTier?: string;
  /** @format uuid */
  defaultApiKeyShareId?: string;
  email?: string;
  hasAccount: boolean;
  hasContact: boolean;
  /** @format uuid */
  id: string;
  language?: string;
  mayChangeAnyRole: boolean;
  mayChangeRole: boolean;
  mayDelete: boolean;
  mayEditAccount: boolean;
  mayEditCard: boolean;
  mayEditContact: boolean;
  mayRemoveMember: boolean;
  meta?: UserMeta;
  name: string;
  registered?: boolean;
  role?: UserRoleRef;
}

export interface UserViewPage {
  items: UserView[];
  total: number;
}

export interface VerifiedPageView {
  /** @format date-time */
  approvedAt?: string;
  callbackUrl: string;
  /** @format date-time */
  createdAt: string;
  /** @format uuid */
  id: string;
  mayApprove: boolean;
  mayDelete: boolean;
  mayEdit: boolean;
  /** @format date-time */
  modifiedAt: string;
  /** @format uuid */
  organisationId: string;
  organisationName: string;
  pageId: string;
  pageUrl: string;
  site: string;
}

export interface VerifiedPageViewPage {
  items: VerifiedPageView[];
  total: number;
}

export interface VerifiedPagesResp {
  pages: VerifiedPageView[];
}

export interface VoiceSampleReq {
  genre: string;
  scenario: string;
  voice: string;
  voiceStyle: string;
}

export interface VoiceSampleResp {
  /** @format byte */
  audio: Blob;
  contentType: string;
  ownVoice: boolean;
  stored: boolean;
}

export interface VoiceSlot {
  choices: VoiceChoice[];
  name: string;
}

export interface Workshop {
  active: boolean;
  aiQualityTier?: string;
  allowGameSharing: boolean;
  /** @format uuid */
  defaultApiKeyShareId?: string;
  /** @format uuid */
  id: string;
  institution?: Institution;
  invites?: UserRoleInvite[];
  isPaused: boolean;
  meta: Meta;
  name: string;
  participants?: WorkshopParticipant[];
  promptConstraints?: string;
  public: boolean;
  publicDescription?: string;
  publicLinks?: PublicWorkshopLink[];
  publicSlug?: string;
  showOtherParticipantsGames: boolean;
  showPublicGames: boolean;
}

export interface WorkshopEvent {
  /** @format uuid */
  gameId?: string;
  /** @format uuid */
  triggeredBy?: string;
  type: EventType;
  /** @format uuid */
  workshopId?: string;
}

export interface WorkshopMeta {
  /** @format date-time */
  createdAt: string;
  /** @format date-time */
  modifiedAt: string;
}

export interface WorkshopParticipant {
  accessToken: string;
  active: boolean;
  gamesCount: number;
  /** @format uuid */
  id: string;
  meta: Meta;
  name: string;
  permanent: boolean;
  role: Role;
  /** @format uuid */
  workshopId: string;
}

export interface WorkshopParticipantView {
  gamesCount: number;
  /** @format uuid */
  id: string;
  meta: ParticipantMeta;
  name: string;
  permanent: boolean;
  role: string;
  /** @format uuid */
  workshopId: string;
}

export interface WorkshopParticipantViewPage {
  items: WorkshopParticipantView[];
  total: number;
}

export interface WorkshopView {
  active: boolean;
  aiQualityTier?: string;
  allowGameSharing: boolean;
  /** @format uuid */
  defaultApiKeyShareId?: string;
  hasParticipants: boolean;
  hasSettings: boolean;
  /** @format uuid */
  id: string;
  institution: NamedRef;
  invites: InviteView[];
  isPaused: boolean;
  mayDelete: boolean;
  mayEditCard: boolean;
  mayEditSettings: boolean;
  mayManageParticipants: boolean;
  meta: WorkshopMeta;
  name: string;
  participants: WorkshopParticipantView[];
  promptConstraints?: string;
  public: boolean;
  publicDescription?: string;
  publicLinks?: PublicWorkshopLink[];
  publicSlug?: string;
  showOtherParticipantsGames: boolean;
  showPublicGames: boolean;
}

export interface WorkshopViewPage {
  items: WorkshopView[];
  total: number;
}

export interface YouthProtectionSuggestion {
  minAge: number;
  text: string;
}

export type QueryParamsType = Record<string | number, any>;
export type ResponseFormat = keyof Omit<Body, "body" | "bodyUsed">;

export interface FullRequestParams extends Omit<RequestInit, "body"> {
  /** set parameter to `true` for call `securityWorker` for this request */
  secure?: boolean;
  /** request path */
  path: string;
  /** content type of request body */
  type?: ContentType;
  /** query params */
  query?: QueryParamsType;
  /** format of response (i.e. response.json() -> format: "json") */
  format?: ResponseFormat;
  /** request body */
  body?: unknown;
  /** base url */
  baseUrl?: string;
  /** request cancellation token */
  cancelToken?: CancelToken;
}

export type RequestParams = Omit<
  FullRequestParams,
  "body" | "method" | "query" | "path"
>;

export interface ApiConfig<SecurityDataType = unknown> {
  baseUrl?: string;
  baseApiParams?: Omit<RequestParams, "baseUrl" | "cancelToken" | "signal">;
  securityWorker?: (
    securityData: SecurityDataType | null,
  ) => Promise<RequestParams | void> | RequestParams | void;
  customFetch?: typeof fetch;
}

export interface HttpResponse<D extends unknown, E extends unknown = unknown>
  extends Response {
  data: D;
  error: E;
}

type CancelToken = Symbol | string | number;

export type ContentType =
  | "application/json"
  | "application/vnd.api+json"
  | "multipart/form-data"
  | "application/x-www-form-urlencoded"
  | "text/plain";

export class HttpClient<SecurityDataType = unknown> {
  public baseUrl: string = "";
  private securityData: SecurityDataType | null = null;
  private securityWorker?: ApiConfig<SecurityDataType>["securityWorker"];
  private abortControllers = new Map<CancelToken, AbortController>();
  private customFetch = (...fetchParams: Parameters<typeof fetch>) =>
    fetch(...fetchParams);

  private baseApiParams: RequestParams = {
    credentials: "same-origin",
    headers: {},
    redirect: "follow",
    referrerPolicy: "no-referrer",
  };

  constructor(apiConfig: ApiConfig<SecurityDataType> = {}) {
    Object.assign(this, apiConfig);
  }

  public setSecurityData = (data: SecurityDataType | null) => {
    this.securityData = data;
  };

  protected encodeQueryParam(key: string, value: any) {
    const encodedKey = encodeURIComponent(key);
    return `${encodedKey}=${encodeURIComponent(typeof value === "number" ? value : `${value}`)}`;
  }

  protected addQueryParam(query: QueryParamsType, key: string) {
    return this.encodeQueryParam(key, query[key]);
  }

  protected addArrayQueryParam(query: QueryParamsType, key: string) {
    const value = query[key];
    return value.map((v: any) => this.encodeQueryParam(key, v)).join("&");
  }

  protected toQueryString(rawQuery?: QueryParamsType): string {
    const query = rawQuery || {};
    const keys = Object.keys(query).filter(
      (key) => "undefined" !== typeof query[key],
    );
    return keys
      .map((key) =>
        Array.isArray(query[key])
          ? this.addArrayQueryParam(query, key)
          : this.addQueryParam(query, key),
      )
      .join("&");
  }

  protected addQueryParams(rawQuery?: QueryParamsType): string {
    const queryString = this.toQueryString(rawQuery);
    return queryString ? `?${queryString}` : "";
  }

  private contentFormatters: Record<ContentType, (input: any) => any> = {
    ["application/json"]: (input: any) =>
      input !== null && (typeof input === "object" || typeof input === "string")
        ? JSON.stringify(input)
        : input,
    ["application/vnd.api+json"]: (input: any) =>
      input !== null && (typeof input === "object" || typeof input === "string")
        ? JSON.stringify(input)
        : input,
    ["text/plain"]: (input: any) =>
      input !== null && typeof input !== "string"
        ? JSON.stringify(input)
        : input,
    ["multipart/form-data"]: (input: any) => {
      if (input instanceof FormData) {
        return input;
      }

      return Object.keys(input || {}).reduce((formData, key) => {
        const property = input[key];
        formData.append(
          key,
          property instanceof Blob
            ? property
            : typeof property === "object" && property !== null
              ? JSON.stringify(property)
              : `${property}`,
        );
        return formData;
      }, new FormData());
    },
    ["application/x-www-form-urlencoded"]: (input: any) =>
      this.toQueryString(input),
  };

  protected mergeRequestParams(
    params1: RequestParams,
    params2?: RequestParams,
  ): RequestParams {
    return {
      ...this.baseApiParams,
      ...params1,
      ...(params2 || {}),
      headers: {
        ...(this.baseApiParams.headers || {}),
        ...(params1.headers || {}),
        ...((params2 && params2.headers) || {}),
      },
    };
  }

  protected createAbortSignal = (
    cancelToken: CancelToken,
  ): AbortSignal | undefined => {
    if (this.abortControllers.has(cancelToken)) {
      const abortController = this.abortControllers.get(cancelToken);
      if (abortController) {
        return abortController.signal;
      }
      return void 0;
    }

    const abortController = new AbortController();
    this.abortControllers.set(cancelToken, abortController);
    return abortController.signal;
  };

  public abortRequest = (cancelToken: CancelToken) => {
    const abortController = this.abortControllers.get(cancelToken);

    if (abortController) {
      abortController.abort();
      this.abortControllers.delete(cancelToken);
    }
  };

  public request = async <T = any, E = any>({
    body,
    secure,
    path,
    type,
    query,
    format,
    baseUrl,
    cancelToken,
    ...params
  }: FullRequestParams): Promise<HttpResponse<T, E>> => {
    const secureParams =
      ((typeof secure === "boolean" ? secure : this.baseApiParams.secure) &&
        this.securityWorker &&
        (await this.securityWorker(this.securityData))) ||
      {};
    const requestParams = this.mergeRequestParams(params, secureParams);
    const queryString = query && this.toQueryString(query);
    const payloadFormatter = this.contentFormatters[type || "application/json"];
    const responseFormat = format || requestParams.format;

    return this.customFetch(
      `${baseUrl || this.baseUrl || ""}${path}${queryString ? `?${queryString}` : ""}`,
      {
        ...requestParams,
        headers: {
          ...(requestParams.headers || {}),
          ...(type && type !== "multipart/form-data"
            ? { "Content-Type": type }
            : {}),
        },
        signal:
          (cancelToken
            ? this.createAbortSignal(cancelToken)
            : requestParams.signal) || null,
        body:
          typeof body === "undefined" || body === null
            ? null
            : payloadFormatter(body),
      },
    ).then(async (response) => {
      const r = response as HttpResponse<T, E>;
      r.data = null as unknown as T;
      r.error = null as unknown as E;

      const responseToParse = responseFormat ? response.clone() : response;
      const data = !responseFormat
        ? r
        : await responseToParse[responseFormat]()
            .then((data) => {
              if (r.ok) {
                r.data = data;
              } else {
                r.error = data;
              }
              return r;
            })
            .catch((e) => {
              r.error = e;
              return r;
            });

      if (cancelToken) {
        this.abortControllers.delete(cancelToken);
      }

      if (!response.ok) throw data;
      return data;
    });
  };
}

/**
 * @title API
 * @version 1.0.0
 */
export class Api<
  SecurityDataType extends unknown,
> extends HttpClient<SecurityDataType> {
  api = {
    /**
     * @description own use of each, the organisation they gave it to, and the sponsored games and share links it pays for.
     *
     * @name GetApiApikeys
     * @summary Lists the caller's keys and every share of them: their
     * @request GET:/api/apikeys
     * @secure
     */
    getApiApikeys: (
      query?: {
        filter?: string[];
        search?: string;
        sortBy?: OwnApiKeySort;
        sortDir?: SortDir;
        limit?: number;
        offset?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<OwnApiKeyViewPage, Error>({
        path: `/api/apikeys`,
        method: "GET",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description own use of it. Platform and key are required; surrounding whitespace is trimmed. Without a name the key is named after its platform and today's date, e.g. "Mistral 12.02.26". The caller's first key becomes their default.
     *
     * @name PostApiApikeysNew
     * @summary Stores a key of the caller's and answers with their
     * @request POST:/api/apikeys/new
     * @secure
     */
    postApiApikeysNew: (data: CreateApiKeyReq, params: RequestParams = {}) =>
      this.request<KeyShareView, Error>({
        path: `/api/apikeys/new`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description owner any share of their key, a head their organisation's, head and staff what draws from it. With cascade=true the owner deletes the key itself, with every share of it. The owner's own use of a key goes only with the key.
     *
     * @name DeleteApiApikeysId
     * @summary Takes a share back, stopping what draws from it: the
     * @request DELETE:/api/apikeys/{id}
     * @secure
     */
    deleteApiApikeysId: (
      id: string,
      query?: {
        cascade?: string;
      },
      params: RequestParams = {},
    ) =>
      this.request<MessageResp, Error>({
        path: `/api/apikeys/${id}`,
        method: "DELETE",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description owner every share of the key (linkedShares).
     *
     * @name GetApiApikeysId
     * @summary Returns a share the caller may see, and for the key's
     * @request GET:/api/apikeys/{id}
     * @secure
     */
    getApiApikeysId: (id: string, params: RequestParams = {}) =>
      this.request<ApiKeyInfoResp, Error>({
        path: `/api/apikeys/${id}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name PatchApiApikeysId
     * @summary Renames a key, for its owner.
     * @request PATCH:/api/apikeys/{id}
     * @secure
     */
    patchApiApikeysId: (
      id: string,
      data: UpdateApiKeyReq,
      params: RequestParams = {},
    ) =>
      this.request<KeyShareView, Error>({
        path: `/api/apikeys/${id}`,
        method: "PATCH",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name PostApiApikeysId
     * @summary Is ShareApiKeyEndpoint at v1's path.
     * @request POST:/api/apikeys/{id}
     * @secure
     */
    postApiApikeysId: (
      id: string,
      data: ShareApiKeyReq,
      params: RequestParams = {},
    ) =>
      this.request<KeyShareView, Error>({
        path: `/api/apikeys/${id}`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description answers the share as the check left it. The check is the smallest call the provider bills, a fraction of a cent on the key: it finds a key the provider rejects (lastErrorCode key_rejected) and an account with no credit left (no_credit) alike, and records the outcome as the key's last usage. A provider that does not answer records nothing: 503 ai_error.
     *
     * @name PostApiApikeysIdCheck
     * @summary Checks a key with its provider, for its owner, and
     * @request POST:/api/apikeys/{id}/check
     * @secure
     */
    postApiApikeysIdCheck: (id: string, params: RequestParams = {}) =>
      this.request<KeyShareView, Error>({
        path: `/api/apikeys/${id}/check`,
        method: "POST",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description play with themselves.
     *
     * @name PutApiApikeysIdDefault
     * @summary Makes a key its owner's default: the one they
     * @request PUT:/api/apikeys/{id}/default
     * @secure
     */
    putApiApikeysIdDefault: (id: string, params: RequestParams = {}) =>
      this.request<KeyShareView, Error>({
        path: `/api/apikeys/${id}/default`,
        method: "PUT",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description of its shares: for the key's owner all of them, for head and staff those their organisation pays for. context=personal needs the owner's own share; context=organization leaves out the owner's personal links.
     *
     * @name GetApiApikeysIdGameShares
     * @summary Lists the share links a key pays for, by one
     * @request GET:/api/apikeys/{id}/game-shares
     * @secure
     */
    getApiApikeysIdGameShares: (
      id: string,
      query?: {
        context?: string;
      },
      params: RequestParams = {},
    ) =>
      this.request<ShareLinkView[], Error>({
        path: `/api/apikeys/${id}/game-shares`,
        method: "GET",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description workshops may play with it, its head and staff sponsor games and make share links with it. A key is shared with an organisation only, never with users or workshops, and never across organisations.
     *
     * @name PostApiApikeysIdShares
     * @summary Gives a key to its owner's own organisation: its
     * @request POST:/api/apikeys/{id}/shares
     * @secure
     */
    postApiApikeysIdShares: (
      id: string,
      data: ShareApiKeyReq,
      params: RequestParams = {},
    ) =>
      this.request<KeyShareView, Error>({
        path: `/api/apikeys/${id}/shares`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description Verifies the ID token from the Auth0 SPA SDK and starts a schemaf session. 404 if Auth0 is not configured on this instance.
     *
     * @name PostApiAuthAuth0Login
     * @summary Sign in with an Auth0 ID token
     * @request POST:/api/auth/auth0/login
     */
    postApiAuthAuth0Login: (
      data: SchemafAuth0LoginRequest,
      params: RequestParams = {},
    ) =>
      this.request<SchemafLoginResponse, Error>({
        path: `/api/auth/auth0/login`,
        method: "POST",
        body: data,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description A name longer than 24 characters is unavailable; a profane one is unavailable and flagged profane.
     *
     * @name GetApiAuthCheckName
     * @summary Checks whether a name is free to register.
     * @request GET:/api/auth/check-name
     */
    getApiAuthCheckName: (
      query: {
        name: string;
      },
      params: RequestParams = {},
    ) =>
      this.request<CheckNameResp, Error>({
        path: `/api/auth/check-name`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),

    /**
     * @description Development only: outside development mode it answers 404. Takes a preset role ("admin-1", "admin-2", "head-1", "head-2", "staff-1", "staff-2", "individual-1", "individual-2", "guest"), starts a session for its account and sets the session cookie. "new-person" signs in a fresh account still registering, as a first Auth0 sign-in does, to try registration without Auth0.
     *
     * @name PostApiAuthDevLogin
     * @summary Signs in as one of the seeded development accounts.
     * @request POST:/api/auth/dev-login
     */
    postApiAuthDevLogin: (data: DevLoginReq, params: RequestParams = {}) =>
      this.request<DevLoginResp, Error>({
        path: `/api/auth/dev-login`,
        method: "POST",
        body: data,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description Takes the participant token whole, or its words from a link, QR code or typed code. Starts a session, sets the session cookie and answers with the session token, for Bearer clients, and the participant's id.
     *
     * @name PostApiAuthGuestLogin
     * @summary Signs a workshop participant in with their token.
     * @request POST:/api/auth/guest-login
     */
    postApiAuthGuestLogin: (data: GuestLoginReq, params: RequestParams = {}) =>
      this.request<GuestLoginResp, Error>({
        path: `/api/auth/guest-login`,
        method: "POST",
        body: data,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description Revokes the current session (if any) and clears the session cookie.
     *
     * @name PostApiAuthLogout
     * @summary End the current session
     * @request POST:/api/auth/logout
     */
    postApiAuthLogout: (params: RequestParams = {}) =>
      this.request<void, Error>({
        path: `/api/auth/logout`,
        method: "POST",
        ...params,
      }),

    /**
     * No description
     *
     * @name PostApiAuthRefresh
     * @summary Get a fresh token for the current session
     * @request POST:/api/auth/refresh
     * @secure
     */
    postApiAuthRefresh: (params: RequestParams = {}) =>
      this.request<SchemafRefreshResponse, Error>({
        path: `/api/auth/refresh`,
        method: "POST",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description answers with their user. An account exists from its first Auth0 sign-in and is unusable until it has a name (at most 24 characters) and an age group; the email is optional. An account registers once (409 afterwards). An email another account has is refused (409 email_belongs_to_other_account). The rights in the answer are those of the registering caller; GET /api/users/me shows the registered account's. With an organisation, the account registers it too and becomes its head instead of an individual. That takes the age group u18 (403 institution_age_not_allowed otherwise), an email and acceptTerms; the organisation is as in POST /api/users/me/organisation. Account and organisation are stored together or not at all.
     *
     * @name PostApiAuthRegister
     * @summary Completes the caller's registration and
     * @request POST:/api/auth/register
     * @secure
     */
    postApiAuthRegister: (
      data: CompleteRegistrationReq,
      params: RequestParams = {},
    ) =>
      this.request<UserView, Error>({
        path: `/api/auth/register`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description Passes the request to the named identity provider and starts a schemaf session. The body is defined by the provider.
     *
     * @name PostApiAuthProviderLogin
     * @summary Sign in with an identity provider
     * @request POST:/api/auth/{provider}/login
     */
    postApiAuthProviderLogin: (
      provider: string,
      data: Record<string, any>,
      params: RequestParams = {},
    ) =>
      this.request<SchemafLoginResponse, Error>({
        path: `/api/auth/${provider}/login`,
        method: "POST",
        body: data,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description Every PUBLIC_* environment variable, keyed by its full name. Read it in the frontend through config() from config.gen.ts.
     *
     * @name GetApiConfig
     * @summary Public frontend config
     * @request GET:/api/config
     */
    getApiConfig: (params: RequestParams = {}) =>
      this.request<Record<string, string>, Error>({
        path: `/api/config`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description newest first, at most 50. Development only: outside development mode it answers 404. For tests and for trying mails without a mail server.
     *
     * @name GetApiDevMails
     * @summary Lists the mails the server sent since it started,
     * @request GET:/api/dev/mails
     */
    getApiDevMails: (params: RequestParams = {}) =>
      this.request<Message[], Error>({
        path: `/api/dev/mails`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description Opened with ?session=<id> to play a session, or ?new&game=<id> or ?new&share=<token> to start one.
     *
     * @name GetApiEnginePlaybackEngine
     * @summary Serves the playback-engine's page.
     * @request GET:/api/engine/playback-engine/
     * @secure
     */
    getApiEnginePlaybackEngine: (params: RequestParams = {}) =>
      this.request<void, Error>({
        path: `/api/engine/playback-engine/`,
        method: "GET",
        secure: true,
        ...params,
      }),

    /**
     * @description Its chunks, fonts, art and language files.
     *
     * @name GetApiEnginePlaybackEngineDirFile
     * @summary Serves a playback-engine file in a folder.
     * @request GET:/api/engine/playback-engine/{dir}/{file}
     * @secure
     */
    getApiEnginePlaybackEngineDirFile: (
      dir: string,
      file: string,
      params: RequestParams = {},
    ) =>
      this.request<void, Error>({
        path: `/api/engine/playback-engine/${dir}/${file}`,
        method: "GET",
        secure: true,
        ...params,
      }),

    /**
     * @description Its scripts, stylesheet and worklet.
     *
     * @name GetApiEnginePlaybackEngineFile
     * @summary Serves a playback-engine file beside its page.
     * @request GET:/api/engine/playback-engine/{file}
     * @secure
     */
    getApiEnginePlaybackEngineFile: (
      file: string,
      params: RequestParams = {},
    ) =>
      this.request<void, Error>({
        path: `/api/engine/playback-engine/${file}`,
        method: "GET",
        secure: true,
        ...params,
      }),

    /**
     * @description A game for a signed-in player, a share token for a guest or a signed-in player. A repeat of the one-time key within ten minutes answers the session the first created, to the same caller only. Counts towards the token lock, as it may look up a share token.
     *
     * @name PostApiEngineSessions
     * @summary Starts a session from a game or a share token.
     * @request POST:/api/engine/sessions
     * @secure
     */
    postApiEngineSessions: (data: CreateRequest, params: RequestParams = {}) =>
      this.request<CreateResponse, Error>({
        path: `/api/engine/sessions`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description A choice outside the offer's caveats is refused.
     *
     * @name PostApiEngineSessionsIdChoose
     * @summary Takes the player's choice among a session's offers.
     * @request POST:/api/engine/sessions/{id}/choose
     * @secure
     */
    postApiEngineSessionsIdChoose: (
      id: string,
      data: ChooseRequest,
      params: RequestParams = {},
    ) =>
      this.request<void, Error>({
        path: `/api/engine/sessions/${id}/choose`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        ...params,
      }),

    /**
     * @description The provider's id stays server-side, keeping a spendable key and a steerable conversation off the client.
     *
     * @name PostApiEngineSessionsIdConnect
     * @summary Brokers the browser's offer to talk to a session's character.
     * @request POST:/api/engine/sessions/{id}/connect
     * @secure
     */
    postApiEngineSessionsIdConnect: (
      id: string,
      data: ConnectRequest,
      params: RequestParams = {},
    ) =>
      this.request<ConnectResponse, Error>({
        path: `/api/engine/sessions/${id}/connect`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description repeats the youth-protection rule. Staff and heads only. 1 repeats it with every call, up to 10 with every tenth; the session keeps the change. The System Message and the opening always carry the rule.
     *
     * @name PostApiEngineSessionsIdConstraintFrequency
     * @summary Sets how often a block of a session
     * @request POST:/api/engine/sessions/{id}/constraint-frequency
     * @secure
     */
    postApiEngineSessionsIdConstraintFrequency: (
      id: string,
      data: ConstraintFrequencyRequest,
      params: RequestParams = {},
    ) =>
      this.request<void, Error>({
        path: `/api/engine/sessions/${id}/constraint-frequency`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        ...params,
      }),

    /**
     * @description The wire is named by its ends and its kind.
     *
     * @name GetApiEngineSessionsIdEdgesFromToKind
     * @summary Answers what one wire of a session has carried.
     * @request GET:/api/engine/sessions/{id}/edges/{from}/{to}/{kind}
     * @secure
     */
    getApiEngineSessionsIdEdgesFromToKind: (
      id: string,
      from: string,
      to: string,
      kind: string,
      params: RequestParams = {},
    ) =>
      this.request<EdgeDetail, Error>({
        path: `/api/engine/sessions/${id}/edges/${from}/${to}/${kind}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiEngineSessionsIdFlowchart
     * @summary Renders a session's running wiring as a mermaid flowchart.
     * @request GET:/api/engine/sessions/{id}/flowchart
     * @secure
     */
    getApiEngineSessionsIdFlowchart: (id: string, params: RequestParams = {}) =>
      this.request<FlowchartResponse, Error>({
        path: `/api/engine/sessions/${id}/flowchart`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiEngineSessionsIdGraph
     * @summary Renders a session's running wiring as text.
     * @request GET:/api/engine/sessions/{id}/graph
     * @secure
     */
    getApiEngineSessionsIdGraph: (id: string, params: RequestParams = {}) =>
      this.request<GraphResponse, Error>({
        path: `/api/engine/sessions/${id}/graph`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description rule its blocks keep to: the one now and the one resolved, where it came from, which blocks keep to it and how often each repeats it. mayEdit says whether the caller may change it: staff and heads, in their own session.
     *
     * @name GetApiEngineSessionsIdGuardrail
     * @summary Answers a session's guardrail, the youth-protection
     * @request GET:/api/engine/sessions/{id}/guardrail
     * @secure
     */
    getApiEngineSessionsIdGuardrail: (id: string, params: RequestParams = {}) =>
      this.request<GuardrailReport, Error>({
        path: `/api/engine/sessions/${id}/guardrail`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Staff and heads only, in their own session: a lab experiment with guardrails, looser or stricter. The blocks that keep to it take the new one from their next call; what was done stays. The resolved guardrail clears the edit.
     *
     * @name PutApiEngineSessionsIdGuardrail
     * @summary Changes a session's guardrail during play.
     * @request PUT:/api/engine/sessions/{id}/guardrail
     * @secure
     */
    putApiEngineSessionsIdGuardrail: (
      id: string,
      data: GuardrailEditRequest,
      params: RequestParams = {},
    ) =>
      this.request<void, Error>({
        path: `/api/engine/sessions/${id}/guardrail`,
        method: "PUT",
        body: data,
        secure: true,
        type: "application/json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiEngineSessionsIdHistory
     * @summary Answers a session's conversation so far, in order.
     * @request GET:/api/engine/sessions/{id}/history
     * @secure
     */
    getApiEngineSessionsIdHistory: (id: string, params: RequestParams = {}) =>
      this.request<SessionEvent[], Error>({
        path: `/api/engine/sessions/${id}/history`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description As the genre accepts; an accepted input counts as one of the session's messages.
     *
     * @name PostApiEngineSessionsIdInput
     * @summary Submits one player action to a session, typed or spoken.
     * @request POST:/api/engine/sessions/{id}/input
     * @secure
     */
    postApiEngineSessionsIdInput: (
      id: string,
      data: InputRequest,
      params: RequestParams = {},
    ) =>
      this.request<void, Error>({
        path: `/api/engine/sessions/${id}/input`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        ...params,
      }),

    /**
     * @description engine saw it: phases, problems, the live conversation's life, pages attaching and leaving. Kept in memory, the latest entries only.
     *
     * @name GetApiEngineSessionsIdJournal
     * @summary Answers what happened in a session so far, as the
     * @request GET:/api/engine/sessions/{id}/journal
     * @secure
     */
    getApiEngineSessionsIdJournal: (id: string, params: RequestParams = {}) =>
      this.request<JournalEntry[], Error>({
        path: `/api/engine/sessions/${id}/journal`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The browser sends binary frames of microphone audio and text frames of typed input, a JSON LiveInput; it receives the session's events back, binary frames for audio and text frames of JSON SessionEvent for everything else. A rejected input comes back as an `error` event.
     *
     * @name GetApiEngineSessionsIdLive
     * @summary Opens the duplex WebSocket a session of a voice genre needs.
     * @request GET:/api/engine/sessions/{id}/live
     * @secure
     */
    getApiEngineSessionsIdLive: (id: string, params: RequestParams = {}) =>
      this.request<void, Error>({
        path: `/api/engine/sessions/${id}/live`,
        method: "GET",
        secure: true,
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiEngineSessionsIdNodesNode
     * @summary Answers what one block of a session is and has been doing.
     * @request GET:/api/engine/sessions/{id}/nodes/{node}
     * @secure
     */
    getApiEngineSessionsIdNodesNode: (
      id: string,
      node: string,
      params: RequestParams = {},
    ) =>
      this.request<NodeDetail, Error>({
        path: `/api/engine/sessions/${id}/nodes/${node}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description before they go to its model, during play: a tuning knob, on dev servers only.
     *
     * @name PostApiEngineSessionsIdNodesNodeBeat
     * @summary Sets how long an observer's transcript fragments wait
     * @request POST:/api/engine/sessions/{id}/nodes/{node}/beat
     * @secure
     */
    postApiEngineSessionsIdNodesNodeBeat: (
      id: string,
      node: string,
      data: BeatRequest,
      params: RequestParams = {},
    ) =>
      this.request<void, Error>({
        path: `/api/engine/sessions/${id}/nodes/${node}/beat`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        ...params,
      }),

    /**
     * @description The thread summarises whatever its fill, the block working meanwhile and paying for the summary; for testing how well compaction keeps a story. A block on no thread is refused.
     *
     * @name PostApiEngineSessionsIdNodesNodeCompact
     * @summary Asks a block's thread to compact now.
     * @request POST:/api/engine/sessions/{id}/nodes/{node}/compact
     * @secure
     */
    postApiEngineSessionsIdNodesNodeCompact: (
      id: string,
      node: string,
      params: RequestParams = {},
    ) =>
      this.request<void, Error>({
        path: `/api/engine/sessions/${id}/nodes/${node}/compact`,
        method: "POST",
        secure: true,
        ...params,
      }),

    /**
     * @description The block, such as an observer, puts it in as a direction the model follows, background it may use, or a line it says. Anyone playing the session may inject. A block that takes none is refused, and so are an unknown kind and blank text.
     *
     * @name PostApiEngineSessionsIdNodesNodeInject
     * @summary Puts text into a session's live conversation through a block.
     * @request POST:/api/engine/sessions/{id}/nodes/{node}/inject
     * @secure
     */
    postApiEngineSessionsIdNodesNodeInject: (
      id: string,
      node: string,
      data: InjectionRequest,
      params: RequestParams = {},
    ) =>
      this.request<void, Error>({
        path: `/api/engine/sessions/${id}/nodes/${node}/inject`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        ...params,
      }),

    /**
     * @description And what the session settled on.
     *
     * @name GetApiEngineSessionsIdOptions
     * @summary Answers what the player of a session may choose from.
     * @request GET:/api/engine/sessions/{id}/options
     * @secure
     */
    getApiEngineSessionsIdOptions: (id: string, params: RequestParams = {}) =>
      this.request<Options, Error>({
        path: `/api/engine/sessions/${id}/options`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description As idling would.
     *
     * @name PostApiEngineSessionsIdPause
     * @summary Lets a session's live conversation go at the player's request.
     * @request POST:/api/engine/sessions/{id}/pause
     * @secure
     */
    postApiEngineSessionsIdPause: (id: string, params: RequestParams = {}) =>
      this.request<void, Error>({
        path: `/api/engine/sessions/${id}/pause`,
        method: "POST",
        secure: true,
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiEngineSessionsIdPrompts
     * @summary Answers the text a session is running on, by block.
     * @request GET:/api/engine/sessions/{id}/prompts
     * @secure
     */
    getApiEngineSessionsIdPrompts: (id: string, params: RequestParams = {}) =>
      this.request<Record<string, string>, Error>({
        path: `/api/engine/sessions/${id}/prompts`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Every player of the session may, to see how the AI's answers change: the youth-protection rule travels with the calls on its own, so no prompt can drop it. The blocks that run on the prompt take the new text from their next call; the session keeps it, also across a resume.
     *
     * @name PutApiEngineSessionsIdPromptsName
     * @summary Changes one of a session's prompts during play.
     * @request PUT:/api/engine/sessions/{id}/prompts/{name}
     * @secure
     */
    putApiEngineSessionsIdPromptsName: (
      id: string,
      name: string,
      data: PromptEditRequest,
      params: RequestParams = {},
    ) =>
      this.request<void, Error>({
        path: `/api/engine/sessions/${id}/prompts/${name}`,
        method: "PUT",
        body: data,
        secure: true,
        type: "application/json",
        ...params,
      }),

    /**
     * @description A reloading page fetches it with the history.
     *
     * @name GetApiEngineSessionsIdState
     * @summary Answers where a session stands now.
     * @request GET:/api/engine/sessions/{id}/state
     * @secure
     */
    getApiEngineSessionsIdState: (id: string, params: RequestParams = {}) =>
      this.request<Snapshot, Error>({
        path: `/api/engine/sessions/${id}/state`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Each event's data is one SessionEvent; its Stream names the kind.
     *
     * @name GetApiEngineSessionsIdStream
     * @summary Streams a session's events as server-sent events.
     * @request GET:/api/engine/sessions/{id}/stream
     * @secure
     */
    getApiEngineSessionsIdStream: (id: string, params: RequestParams = {}) =>
      this.request<SessionEvent, Error>({
        path: `/api/engine/sessions/${id}/stream`,
        method: "GET",
        secure: true,
        ...params,
      }),

    /**
     * @description Such as pictures off; on only as far as the session's offer allows.
     *
     * @name PostApiEngineSessionsIdToggles
     * @summary Switches a feature of a session during play.
     * @request POST:/api/engine/sessions/{id}/toggles
     * @secure
     */
    postApiEngineSessionsIdToggles: (
      id: string,
      data: ToggleRequest,
      params: RequestParams = {},
    ) =>
      this.request<void, Error>({
        path: `/api/engine/sessions/${id}/toggles`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiEngineSessionsIdTopology
     * @summary Answers a session's wiring as data, for a view that draws it.
     * @request GET:/api/engine/sessions/{id}/topology
     * @secure
     */
    getApiEngineSessionsIdTopology: (id: string, params: RequestParams = {}) =>
      this.request<Topology, Error>({
        path: `/api/engine/sessions/${id}/topology`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The details stored for the 5xx answered with this id: open in development mode, in production only to callers the app allows (app.CanSeeErrorDetails). 404 when the id is unknown or expired (kept: newest 1000, at most 24 hours, per process).
     *
     * @name GetApiErrorsId
     * @summary Details of a server error
     * @request GET:/api/errors/{id}
     * @secure
     */
    getApiErrorsId: (id: string, params: RequestParams = {}) =>
      this.request<SchemafErrorDetails, Error>({
        path: `/api/errors/${id}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description total. It narrows what the caller may see, never widens it; the order uses card fields only.
     *
     * @name GetApiGames
     * @summary Lists a page of the games the caller can see, and the
     * @request GET:/api/games
     * @secure
     */
    getApiGames: (
      query?: {
        filter?: GameFilter[];
        search?: string;
        sortBy?: GameSort;
        sortDir?: SortDir;
        limit?: number;
        offset?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<GameViewPage, Error>({
        path: `/api/games`,
        method: "GET",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description new game starts with and what the caller may do with it. Nothing is stored; the client fills it in and creates the game with CreateGameEndpoint.
     *
     * @name GetApiGamesNew
     * @summary Returns the template for a game the caller creates: what a
     * @request GET:/api/games/new
     * @secure
     */
    getApiGamesNew: (params: RequestParams = {}) =>
      this.request<GameTemplate, Error>({
        path: `/api/games/new`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description template, and answers with it as stored. Only the name is required, at most 70 characters; making it public takes the template's mayPublish. In a workshop, the game belongs to it.
     *
     * @name PostApiGamesNew
     * @summary Creates a game owned by the caller from the filled-in
     * @request POST:/api/games/new
     * @secure
     */
    postApiGamesNew: (data: CreateGameReq, params: RequestParams = {}) =>
      this.request<GameView, Error>({
        path: `/api/games/new`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description not, and answers the one point most worth improving: the field it is about (title, description, scenario, gameStart, imageStyle, voiceStyle, statusFields), empty when there is nothing to fix; missing when a required field is empty, which the client says in its own words; else one short sentence in the caller's language. Nothing is stored, and nothing blocks saving. The caller pays with their own keys, as in the editor: workshop, organisation, own, site; never a game's sponsor. None: 400 no_api_key; a key its provider rejects: 400 invalid_api_key; a provider that fails or takes too long: 503 ai_error.
     *
     * @name PostApiGamesReview
     * @summary Reviews a game's form as the editor holds it, saved or
     * @request POST:/api/games/review
     * @secure
     */
    postApiGamesReview: (data: ReviewGameReq, params: RequestParams = {}) =>
      this.request<ReviewGameResp, Error>({
        path: `/api/games/review`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description takes mayDelete (403 otherwise).
     *
     * @name DeleteApiGamesId
     * @summary Deletes a game and answers with it as it was. Deleting
     * @request DELETE:/api/games/{id}
     * @secure
     */
    deleteApiGamesId: (id: string, params: RequestParams = {}) =>
      this.request<GameView, Error>({
        path: `/api/games/${id}`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiGamesId
     * @summary Returns a game the caller can see.
     * @request GET:/api/games/{id}
     * @secure
     */
    getApiGamesId: (id: string, params: RequestParams = {}) =>
      this.request<GameView, Error>({
        path: `/api/games/${id}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description game as stored. Send what changes; a field left out stays as it is, and sending the whole game back is fine: only what differs counts. Changing a section the caller may not edit (mayEditCard, mayEditContent), or making the game public without mayPublish, refuses the whole request (403). The name is at most 70 characters. Workshop, sponsor and creator are not changed here.
     *
     * @name PostApiGamesId
     * @summary Changes a game's card and content and answers with the
     * @request POST:/api/games/{id}
     * @secure
     */
    postApiGamesId: (
      id: string,
      data: UpdateGameReq,
      params: RequestParams = {},
    ) =>
      this.request<GameView, Error>({
        path: `/api/games/${id}`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description caller's plays of a game they may read.
     *
     * @name GetApiGamesIdApiKeyStatus
     * @summary Answers whether a key would pay for the
     * @request GET:/api/games/{id}/api-key-status
     * @secure
     */
    getApiGamesIdApiKeyStatus: (id: string, params: RequestParams = {}) =>
      this.request<GameApiKeyStatusResp, Error>({
        path: `/api/games/${id}/api-key-status`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description caller's plays of a game they may read, in the order they are tried.
     *
     * @name GetApiGamesIdAvailableKeys
     * @summary Lists the keys that would pay for the
     * @request GET:/api/games/{id}/available-keys
     * @secure
     */
    getApiGamesIdAvailableKeys: (id: string, params: RequestParams = {}) =>
      this.request<AvailableKey[], Error>({
        path: `/api/games/${id}/available-keys`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description A .cgl file is a zip. It holds game.yaml, the game file as the YAML export writes it, and, when the game has a cover, cover.jpg and cover.sig, the signature an import asks for. It takes seeing the game's content (hasContent; 403 otherwise).
     *
     * @name GetApiGamesIdCgl
     * @summary Exports a game as a .cgl file with its cover.
     * @request GET:/api/games/{id}/cgl
     * @secure
     */
    getApiGamesIdCgl: (id: string, params: RequestParams = {}) =>
      this.request<void, Error>({
        path: `/api/games/${id}/cgl`,
        method: "GET",
        secure: true,
        ...params,
      }),

    /**
     * @description and answers with the copy. It takes mayCopy (403 otherwise); the copy has the game's card and content, not its images.
     *
     * @name PostApiGamesIdClone
     * @summary Copies a game into a new private game of the caller's
     * @request POST:/api/games/{id}/clone
     * @secure
     */
    postApiGamesIdClone: (id: string, params: RequestParams = {}) =>
      this.request<GameView, Error>({
        path: `/api/games/${id}/clone`,
        method: "POST",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description game: 404 for a game they may not see or one without a cover. With ?v= (the game's coverAt) the answer may be cached for good, as a new cover comes with a new v.
     *
     * @name GetApiGamesIdCover
     * @summary Serves a game's cover picture to whoever may see the
     * @request GET:/api/games/{id}/cover
     * @secure
     */
    getApiGamesIdCover: (id: string, params: RequestParams = {}) =>
      this.request<void, Error>({
        path: `/api/games/${id}/cover`,
        method: "GET",
        secure: true,
        ...params,
      }),

    /**
     * @description had, and answers with the game as it now is (coverAt set). It takes mayEditCard (403 otherwise) and blocks until the picture is made. The caller pays as for a session of the game: the first key of theirs that draws pictures, workshop, sponsor, organisation, own, site. None: 400 no_api_key; a key its provider rejects: 400 invalid_api_key; a provider that fails: 503 ai_error. The hint, at most 300 characters, says what the cover should show; empty leaves it to the game.
     *
     * @name PostApiGamesIdCover
     * @summary Draws a new cover for a game, replacing the one it
     * @request POST:/api/games/{id}/cover
     * @secure
     */
    postApiGamesIdCover: (
      id: string,
      data: MakeGameCoverReq,
      params: RequestParams = {},
    ) =>
      this.request<GameView, Error>({
        path: `/api/games/${id}/cover`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description It answers with the game as it now is (coverAt set). The body is the picture, PNG or JPEG, at most 1 MB (413 file_too_large otherwise), and the X-Cover-Signature header the file's cover.sig; one without a fitting signature is 422 cover_unsigned, one that is no picture 400 not_a_picture. It takes mayEditCard (403 otherwise). The picture is checked against the youth-protection rule for the youngest players, as a drawn cover keeps to it, and the caller pays for the check as for drawing a cover: no key, 400 no_api_key; a key its provider rejects, 400 invalid_api_key; a provider that fails, 503 ai_error. A picture that does not keep to the rule is 422 cover_refused, and the game keeps the cover it had.
     *
     * @name PutApiGamesIdCover
     * @summary Takes a game's cover from its .cgl file.
     * @request PUT:/api/games/{id}/cover
     * @secure
     */
    putApiGamesIdCover: (id: string, params: RequestParams = {}) =>
      this.request<void, Error>({
        path: `/api/games/${id}/cover`,
        method: "PUT",
        secure: true,
        ...params,
      }),

    /**
     * @description the game.
     *
     * @name GetApiGamesIdEmbed
     * @summary Reads a game's embed link, for whoever may share
     * @request GET:/api/games/{id}/embed
     * @secure
     */
    getApiGamesIdEmbed: (id: string, params: RequestParams = {}) =>
      this.request<GameEmbedResp, Error>({
        path: `/api/games/${id}/embed`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description for whoever may share the game. Its token never changes: off keeps it, on brings back the same snippet.
     *
     * @name PutApiGamesIdEmbed
     * @summary Sets a game's embed link, making it the first time,
     * @request PUT:/api/games/{id}/embed
     * @secure
     */
    putApiGamesIdEmbed: (
      id: string,
      data: SetGameEmbedReq,
      params: RequestParams = {},
    ) =>
      this.request<GameEmbedResp, Error>({
        path: `/api/games/${id}/embed`,
        method: "PUT",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description answers the same whatever the game: it gives nothing away.
     *
     * @name DeleteApiGamesIdFavourite
     * @summary Removes a game from the caller's favorites. It
     * @request DELETE:/api/games/{id}/favourite
     * @secure
     */
    deleteApiGamesIdFavourite: (id: string, params: RequestParams = {}) =>
      this.request<FavouriteResp, Error>({
        path: `/api/games/${id}/favourite`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description It takes reading the game (404 otherwise); marking twice changes nothing.
     *
     * @name PostApiGamesIdFavourite
     * @summary Marks a game as one of the caller's favorites.
     * @request POST:/api/games/{id}/favourite
     * @secure
     */
    postApiGamesIdFavourite: (id: string, params: RequestParams = {}) =>
      this.request<FavouriteResp, Error>({
        path: `/api/games/${id}/favourite`,
        method: "POST",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description see: those they made, of their game, paid by their keys or their organisation's, and their workshop's. A game they may not read is 404.
     *
     * @name GetApiGamesIdPrivateShare
     * @summary Lists a game's share links the caller may
     * @request GET:/api/games/{id}/private-share
     * @secure
     */
    getApiGamesIdPrivateShare: (id: string, params: RequestParams = {}) =>
      this.request<PrivateShareStatusResp, Error>({
        path: `/api/games/${id}/private-share`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description read; another's game they may not read is 404.
     *
     * @name GetApiGamesIdSessions
     * @summary Lists the caller's sessions of a game they may
     * @request GET:/api/games/{id}/sessions
     * @secure
     */
    getApiGamesIdSessions: (id: string, params: RequestParams = {}) =>
      this.request<SessionView[], Error>({
        path: `/api/games/${id}/sessions`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description share: paid by a key share they may draw from (sponsorKeyShareId), or by the workshop's key (workshopId) for the workshop's head and staff, and for those in it while it allows sharing. A workshop has one link per game; an existing one is answered. maxSessions limits the plays; null: unlimited.
     *
     * @name PostApiGamesIdShares
     * @summary Makes a share link for a game the caller may
     * @request POST:/api/games/{id}/shares
     * @secure
     */
    postApiGamesIdShares: (
      id: string,
      data: CreateGameShareReq,
      params: RequestParams = {},
    ) =>
      this.request<ShareLinkView, Error>({
        path: `/api/games/${id}/shares`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description maker, the game's owner, the key's owner, and head and staff when their organisation's key pays.
     *
     * @name DeleteApiGamesIdSharesShareId
     * @summary Stops a share link: its guests go with it. For its
     * @request DELETE:/api/games/{id}/shares/{shareId}
     * @secure
     */
    deleteApiGamesIdSharesShareId: (
      id: string,
      shareId: string,
      params: RequestParams = {},
    ) =>
      this.request<MessageResp, Error>({
        path: `/api/games/${id}/shares/${shareId}`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description may stop it. Both are set: a null maxSessions makes it unlimited, a null aiQualityTier plays at the source's default.
     *
     * @name PatchApiGamesIdSharesShareId
     * @summary Sets a share link's limit and tier, for those who
     * @request PATCH:/api/games/{id}/shares/{shareId}
     * @secure
     */
    patchApiGamesIdSharesShareId: (
      id: string,
      shareId: string,
      data: UpdateGameShareReq,
      params: RequestParams = {},
    ) =>
      this.request<ShareLinkView, Error>({
        path: `/api/games/${id}/shares/${shareId}`,
        method: "PATCH",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description the game, the key's owner, and head and staff when their organisation's key pays.
     *
     * @name DeleteApiGamesIdSponsor
     * @summary Stops a game's sponsorship: for those who edit
     * @request DELETE:/api/games/{id}/sponsor
     * @secure
     */
    deleteApiGamesIdSponsor: (id: string, params: RequestParams = {}) =>
      this.request<GameView, Error>({
        path: `/api/games/${id}/sponsor`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description replacing what paid before: for those who edit the game, with a share they may draw from. A key known to fail is refused.
     *
     * @name PutApiGamesIdSponsor
     * @summary Makes a key share pay for a game's public plays,
     * @request PUT:/api/games/{id}/sponsor
     * @secure
     */
    putApiGamesIdSponsor: (
      id: string,
      data: GameSponsorReq,
      params: RequestParams = {},
    ) =>
      this.request<GameView, Error>({
        path: `/api/games/${id}/sponsor`,
        method: "PUT",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description takes seeing the game's content (hasContent; 403 otherwise).
     *
     * @name GetApiGamesIdYaml
     * @summary Exports a game as a game file, YAML by default. It
     * @request GET:/api/games/{id}/yaml
     * @secure
     */
    getApiGamesIdYaml: (id: string, params: RequestParams = {}) =>
      this.request<GameFile, Error>({
        path: `/api/games/${id}/yaml`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description answers with the game as stored. The file replaces the game's card and content; its id is ignored. It takes the same rights as an update (mayEditCard, mayEditContent).
     *
     * @name PutApiGamesIdYaml
     * @summary Updates a game from a game file, YAML or JSON, and
     * @request PUT:/api/games/{id}/yaml
     * @secure
     */
    putApiGamesIdYaml: (
      id: string,
      data: UpdateGameYAMLReq,
      params: RequestParams = {},
    ) =>
      this.request<GameView, Error>({
        path: `/api/games/${id}/yaml`,
        method: "PUT",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description With the toggles each needs, for the game editor.
     *
     * @name GetApiGenres
     * @summary Lists every genre the engine runs.
     * @request GET:/api/genres
     * @secure
     */
    getApiGenres: (params: RequestParams = {}) =>
      this.request<GenreInfo[], Error>({
        path: `/api/genres`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description "status" is "ok" or "error", "db" and every registered health check are "ok" or "error: …". 503 (with the same body) when anything is unhealthy.
     *
     * @name GetApiHealth
     * @summary Health check
     * @request GET:/api/health
     */
    getApiHealth: (params: RequestParams = {}) =>
      this.request<Record<string, string>, Record<string, string> | Error>({
        path: `/api/health`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description list, and the total: an admin all of them, a head or staff their own.
     *
     * @name GetApiInstitutions
     * @summary Lists a page of the organisations the caller may
     * @request GET:/api/institutions
     * @secure
     */
    getApiInstitutions: (
      query?: {
        filter?: OrganisationFilter[];
        search?: string;
        sortBy?: OrganisationSort;
        sortDir?: SortDir;
        limit?: number;
        offset?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<OrganisationViewPage, Error>({
        path: `/api/institutions`,
        method: "GET",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The name is required; operator and address are optional. An organisation registers itself through POST /api/auth/register instead.
     *
     * @name PostApiInstitutions
     * @summary Creates an organisation for others: the admin's.
     * @request POST:/api/institutions
     * @secure
     */
    postApiInstitutions: (
      data: CreateInstitutionReq,
      params: RequestParams = {},
    ) =>
      this.request<OrganisationView, Error>({
        path: `/api/institutions`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description workshops and the guests in them; its members stay, as individuals. Suspending keeps its data instead (POST /api/institutions/{id}/suspension).
     *
     * @name DeleteApiInstitutionsId
     * @summary Deletes an organisation for good, with its
     * @request DELETE:/api/institutions/{id}
     * @secure
     */
    deleteApiInstitutionsId: (id: string, params: RequestParams = {}) =>
      this.request<MessageResp, Error>({
        path: `/api/institutions/${id}`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description own, as a member or a guest in one of its workshops; any, for an admin. Others get 404.
     *
     * @name GetApiInstitutionsId
     * @summary Returns an organisation the caller may read: their
     * @request GET:/api/institutions/{id}
     * @secure
     */
    getApiInstitutionsId: (id: string, params: RequestParams = {}) =>
      this.request<OrganisationView, Error>({
        path: `/api/institutions/${id}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description youth-protection restriction, or both, and answers with it as stored. Only what is given changes; an empty promptConstraints removes the restriction, so the site's rule applies. Renaming takes mayEditCard, the restriction mayEditSettings; a request with a part the caller may not change is refused as a whole.
     *
     * @name PatchApiInstitutionsId
     * @summary Changes an organisation: its name, its
     * @request PATCH:/api/institutions/{id}
     * @secure
     */
    patchApiInstitutionsId: (
      id: string,
      data: UpdateInstitutionReq,
      params: RequestParams = {},
    ) =>
      this.request<OrganisationView, Error>({
        path: `/api/institutions/${id}`,
        method: "PATCH",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description its head and staff.
     *
     * @name GetApiInstitutionsIdApikeys
     * @summary Lists the keys given to an organisation: for
     * @request GET:/api/institutions/{id}/apikeys
     * @secure
     */
    getApiInstitutionsIdApikeys: (
      id: string,
      query?: {
        filter?: string[];
        search?: string;
        sortBy?: OrganisationKeySort;
        sortDir?: SortDir;
        limit?: number;
        offset?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<KeyShareViewPage, Error>({
        path: `/api/institutions/${id}/apikeys`,
        method: "GET",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description declaration and answers with the organisation as its heads consent for it. For a head of that organisation (403 otherwise). The organisation is suspended until a head consents again; one without a standing consent has none to withdraw (409). The consent log keeps the consent and its withdrawal.
     *
     * @name DeleteApiInstitutionsIdDeclarationConsent
     * @summary Withdraws a head's organisation's consent to the
     * @request DELETE:/api/institutions/{id}/declaration-consent
     * @secure
     */
    deleteApiInstitutionsIdDeclarationConsent: (
      id: string,
      params: RequestParams = {},
    ) =>
      this.request<OrganisationConsentView, Error>({
        path: `/api/institutions/${id}/declaration-consent`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description consent for it. For a head of that organisation (403 otherwise), also while it is suspended: its consent status (pending while it has no consent to the current declaration, which suspends it) and the operator, address and youth-protection restriction a consent confirms.
     *
     * @name GetApiInstitutionsIdDeclarationConsent
     * @summary Returns an organisation as its heads
     * @request GET:/api/institutions/{id}/declaration-consent
     * @secure
     */
    getApiInstitutionsIdDeclarationConsent: (
      id: string,
      params: RequestParams = {},
    ) =>
      this.request<OrganisationConsentView, Error>({
        path: `/api/institutions/${id}/declaration-consent`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description declaration for their organisation and answers with the organisation as its heads consent for it. For a head of that organisation (403 otherwise), also while it is suspended; one head's consent covers the organisation and lifts its suspension for want of a consent (an admin's stays). It stores the operator, the address and the youth-protection restriction (at most 200 characters) given with it, and confirms the consent by mail, with a copy to the operator.
     *
     * @name PostApiInstitutionsIdDeclarationConsent
     * @summary Records a head's consent to the current
     * @request POST:/api/institutions/{id}/declaration-consent
     * @secure
     */
    postApiInstitutionsIdDeclarationConsent: (
      id: string,
      data: ConsentToDeclarationReq,
      params: RequestParams = {},
    ) =>
      this.request<OrganisationConsentView, Error>({
        path: `/api/institutions/${id}/declaration-consent`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description organisation's members play with for free: one of the keys given to it. A null shareId clears it.
     *
     * @name PatchApiInstitutionsIdFreeUseKey
     * @summary Sets or clears the key an
     * @request PATCH:/api/institutions/{id}/free-use-key
     * @secure
     */
    patchApiInstitutionsIdFreeUseKey: (
      id: string,
      data: SetInstitutionFreeUseKeyReq,
      params: RequestParams = {},
    ) =>
      this.request<OrganisationView, Error>({
        path: `/api/institutions/${id}/free-use-key`,
        method: "PATCH",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description (head and staff), and the total: for its members and the admin. Anyone else gets an empty list; an organisation the caller may not read, 404.
     *
     * @name GetApiInstitutionsIdMembers
     * @summary Lists a page of an organisation's members
     * @request GET:/api/institutions/{id}/members
     * @secure
     */
    getApiInstitutionsIdMembers: (
      id: string,
      query?: {
        filter?: string[];
        search?: string;
        sortBy?: MemberSort;
        sortDir?: SortDir;
        limit?: number;
        offset?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<UserViewPage, Error>({
        path: `/api/institutions/${id}/members`,
        method: "GET",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description keep their account, as an individual. A head removes members and guests, staff remove guests, a member leaves, an admin removes anyone; the last head stays (409 last_head).
     *
     * @name DeleteApiInstitutionsIdMembersUserId
     * @summary Removes a user from an organisation; they
     * @request DELETE:/api/institutions/{id}/members/{userID}
     * @secure
     */
    deleteApiInstitutionsIdMembersUserId: (
      id: string,
      userId: string,
      params: RequestParams = {},
    ) =>
      this.request<MessageResp, Error>({
        path: `/api/institutions/${id}/members/${userId}`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description answers with the suspension, now empty. Admins only (403 otherwise); an organisation without a consent to the current declaration stays suspended until a head consents.
     *
     * @name DeleteApiInstitutionsIdSuspension
     * @summary Lifts an admin's suspension of an organisation and
     * @request DELETE:/api/institutions/{id}/suspension
     * @secure
     */
    deleteApiInstitutionsIdSuspension: (
      id: string,
      params: RequestParams = {},
    ) =>
      this.request<OrganisationSuspensionView, Error>({
        path: `/api/institutions/${id}/suspension`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description suspension. Admins only (403 otherwise). Its heads, staff and the guests in its workshops are refused everywhere but GET /api/users/me (which reports the suspension) and logout; its share links and public workshop pages stop. Nothing is deleted; lifting the suspension reverses it. The reason is an optional note for admins.
     *
     * @name PostApiInstitutionsIdSuspension
     * @summary Suspends an organisation and answers with the
     * @request POST:/api/institutions/{id}/suspension
     * @secure
     */
    postApiInstitutionsIdSuspension: (
      id: string,
      data: SuspendOrganisationReq,
      params: RequestParams = {},
    ) =>
      this.request<OrganisationSuspensionView, Error>({
        path: `/api/institutions/${id}/suspension`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description head and staff.
     *
     * @name GetApiInstitutionsIdVerifiedPages
     * @summary Lists an organisation's verified pages, for its
     * @request GET:/api/institutions/{id}/verified-pages
     * @secure
     */
    getApiInstitutionsIdVerifiedPages: (
      id: string,
      params: RequestParams = {},
    ) =>
      this.request<VerifiedPagesResp, Error>({
        path: `/api/institutions/${id}/verified-pages`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description with a new page ID, for its head and staff.
     *
     * @name PostApiInstitutionsIdVerifiedPages
     * @summary Registers a verified page for an organisation,
     * @request POST:/api/institutions/{id}/verified-pages
     * @secure
     */
    postApiInstitutionsIdVerifiedPages: (
      id: string,
      data: CreateVerifiedPageReq,
      params: RequestParams = {},
    ) =>
      this.request<VerifiedPageView, Error>({
        path: `/api/institutions/${id}/verified-pages`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description account or email: what they may accept or decline.
     *
     * @name GetApiInvites
     * @summary Lists the pending invites addressed to the caller, by
     * @request GET:/api/invites
     * @secure
     */
    getApiInvites: (params: RequestParams = {}) =>
      this.request<InviteCardView[], Error>({
        path: `/api/invites`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description and the total: for the admin, all of them.
     *
     * @name GetApiInvitesAll
     * @summary Lists a page of every invite the caller may list,
     * @request GET:/api/invites/all
     * @secure
     */
    getApiInvitesAll: (
      query?: {
        filter?: string[];
        search?: string;
        sortBy?: InviteSort;
        sortDir?: SortDir;
        limit?: number;
        offset?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<InviteCardViewPage, Error>({
        path: `/api/invites/all`,
        method: "GET",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description an organisation as head or staff (staff by default): accepting, they become it. Head and staff invite into their organisation, only heads as head; the admin into any.
     *
     * @name PostApiInvitesInstitution
     * @summary Invites an account, by id or email, into
     * @request POST:/api/invites/institution
     * @secure
     */
    postApiInvitesInstitution: (
      data: CreateInstitutionInviteReq,
      params: RequestParams = {},
    ) =>
      this.request<InviteCardView, Error>({
        path: `/api/invites/institution`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description the caller may list, and the total: its head and staff (staff without the invites to become head), and the admin.
     *
     * @name GetApiInvitesInstitutionInstitutionId
     * @summary Lists a page of an organisation's invites
     * @request GET:/api/invites/institution/{institutionId}
     * @secure
     */
    getApiInvitesInstitutionInstitutionId: (
      institutionId: string,
      query?: {
        filter?: string[];
        search?: string;
        sortBy?: InviteSort;
        sortDir?: SortDir;
        limit?: number;
        offset?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<InviteCardViewPage, Error>({
        path: `/api/invites/institution/${institutionId}`,
        method: "GET",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description a new one. Whoever follows it becomes a guest there, or, a member of an organisation, works in it.
     *
     * @name PostApiInvitesWorkshop
     * @summary Returns a workshop's link: its usable one, or
     * @request POST:/api/invites/workshop
     * @secure
     */
    postApiInvitesWorkshop: (
      data: CreateWorkshopInviteReq,
      params: RequestParams = {},
    ) =>
      this.request<InviteCardView, Error>({
        path: `/api/invites/workshop`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description into a workshop: accepting, they keep their role and work in it.
     *
     * @name PostApiInvitesWorkshopEmail
     * @summary Invites the account with this email
     * @request POST:/api/invites/workshop/email
     * @secure
     */
    postApiInvitesWorkshopEmail: (
      data: CreateWorkshopEmailInviteReq,
      params: RequestParams = {},
    ) =>
      this.request<InviteCardView, Error>({
        path: `/api/invites/workshop/email`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description its link's token, for anyone who has it, signed in or not. Repeated guesses at tokens are slowed down (429).
     *
     * @name GetApiInvitesIdOrToken
     * @summary Returns an invite: by id, for those who may read it; by
     * @request GET:/api/invites/{idOrToken}
     * @secure
     */
    getApiInvitesIdOrToken: (idOrToken: string, params: RequestParams = {}) =>
      this.request<InviteCardView, Error>({
        path: `/api/invites/${idOrToken}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Following a link without an account joins its workshop as a new guest, signed in at once. A member of an organisation keeps their role and works in the workshop; anyone else takes the invite's role. Only a pending invite is accepted (409). Repeated guesses at tokens are slowed down (429).
     *
     * @name PostApiInvitesIdOrTokenAccept
     * @summary Accepts an invite, by its id or its link's token.
     * @request POST:/api/invites/{idOrToken}/accept
     * @secure
     */
    postApiInvitesIdOrTokenAccept: (
      idOrToken: string,
      params: RequestParams = {},
    ) =>
      this.request<AcceptInviteResp, Error>({
        path: `/api/invites/${idOrToken}/accept`,
        method: "POST",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description and staff (staff not the invites to become head), and the admin. An answered invite stays (409).
     *
     * @name DeleteApiInvitesId
     * @summary Deletes a pending invite: for its organisation's head
     * @request DELETE:/api/invites/{id}
     * @secure
     */
    deleteApiInvitesId: (id: string, params: RequestParams = {}) =>
      this.request<MessageResp, Error>({
        path: `/api/invites/${id}`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name PostApiInvitesIdDecline
     * @summary Declines a pending invite addressed to the caller.
     * @request POST:/api/invites/{id}/decline
     * @secure
     */
    postApiInvitesIdDecline: (id: string, params: RequestParams = {}) =>
      this.request<MessageResp, Error>({
        path: `/api/invites/${id}/decline`,
        method: "POST",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiLanguages
     * @summary Lists the languages the application is available in.
     * @request GET:/api/languages
     */
    getApiLanguages: (params: RequestParams = {}) =>
      this.request<Language[], Error>({
        path: `/api/languages`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description (e.g. "fr", "de", "no"), as a JSON object.
     *
     * @name GetApiLanguagesCode
     * @summary Returns the translations of a language, by its code
     * @request GET:/api/languages/{code}
     */
    getApiLanguagesCode: (code: string, params: RequestParams = {}) =>
      this.request<Record<string, any>, Error>({
        path: `/api/languages/${code}`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description Public. The current version of the Einverständniserklärung: its binding German text (Markdown, title in bold on the first line), the sentence a person ticks to agree, version, date and SHA-256 hash; the suggested youth-protection restrictions; the site's restriction for 13–17 year olds without parental consent, which an organisation's form starts with; and the longest restriction an organisation may set.
     *
     * @name GetApiLegalInstitutionDeclaration
     * @summary Returns the declaration organisations consent to.
     * @request GET:/api/legal/institution-declaration
     */
    getApiLegalInstitutionDeclaration: (params: RequestParams = {}) =>
      this.request<DeclarationResp, Error>({
        path: `/api/legal/institution-declaration`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description important facts that are true now and wait for their acknowledgement.
     *
     * @name GetApiNotifications
     * @summary Answers the caller's notifications: the
     * @request GET:/api/notifications
     * @secure
     */
    getApiNotifications: (params: RequestParams = {}) =>
      this.request<NotificationsResp, Error>({
        path: `/api/notifications`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description notifications and answers those that remain.
     *
     * @name PostApiNotificationsAcknowledge
     * @summary Acknowledges one of the caller's
     * @request POST:/api/notifications/acknowledge
     * @secure
     */
    postApiNotificationsAcknowledge: (
      data: AcknowledgeNotificationReq,
      params: RequestParams = {},
    ) =>
      this.request<NotificationsResp, Error>({
        path: `/api/notifications/acknowledge`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description caller has, each up to the entry they could see; it answers those that remain.
     *
     * @name PostApiNotificationsAcknowledgeAll
     * @summary Acknowledges every notification the
     * @request POST:/api/notifications/acknowledge-all
     * @secure
     */
    postApiNotificationsAcknowledgeAll: (params: RequestParams = {}) =>
      this.request<NotificationsResp, Error>({
        path: `/api/notifications/acknowledge-all`,
        method: "POST",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description With their toggles, playable genres and tiers.
     *
     * @name GetApiPlatforms
     * @summary Lists the engine's AI platforms.
     * @request GET:/api/platforms
     * @secure
     */
    getApiPlatforms: (params: RequestParams = {}) =>
      this.request<PlatformInfo[], Error>({
        path: `/api/platforms`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description welcome screen: its name, description and how many plays are left.
     *
     * @name GetApiPlayTokenInfo
     * @summary Returns the game a share token plays, for the
     * @request GET:/api/play/{token}/info
     */
    getApiPlayTokenInfo: (token: string, params: RequestParams = {}) =>
      this.request<GuestGameInfoResp, Error>({
        path: `/api/play/${token}/info`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description and public games. An unknown slug and a switched-off page both answer 404.
     *
     * @name GetApiPublicWorkshopsSlug
     * @summary Returns a public workshop's name, description
     * @request GET:/api/public/workshops/{slug}
     */
    getApiPublicWorkshopsSlug: (slug: string, params: RequestParams = {}) =>
      this.request<PublicWorkshopPage, Error>({
        path: `/api/public/workshops/${slug}`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description The file is the one GetGameCglEndpoint answers, for a game listed on a public workshop page.
     *
     * @name GetApiPublicWorkshopsSlugGamesIdCgl
     * @summary Exports a public workshop's game as a .cgl file.
     * @request GET:/api/public/workshops/{slug}/games/{id}/cgl
     */
    getApiPublicWorkshopsSlugGamesIdCgl: (
      slug: string,
      id: string,
      params: RequestParams = {},
    ) =>
      this.request<void, Error>({
        path: `/api/public/workshops/${slug}/games/${id}/cgl`,
        method: "GET",
        ...params,
      }),

    /**
     * @description page as a game file, YAML by default.
     *
     * @name GetApiPublicWorkshopsSlugGamesIdYaml
     * @summary Exports a game listed on a public workshop
     * @request GET:/api/public/workshops/{slug}/games/{id}/yaml
     */
    getApiPublicWorkshopsSlugGamesIdYaml: (
      slug: string,
      id: string,
      params: RequestParams = {},
    ) =>
      this.request<GameFile, Error>({
        path: `/api/public/workshops/${slug}/games/${id}/yaml`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name PostApiRestart
     * @summary Has the server restart; admins only.
     * @request POST:/api/restart
     * @secure
     */
    postApiRestart: (params: RequestParams = {}) =>
      this.request<string, Error>({
        path: `/api/restart`,
        method: "POST",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiRoles
     * @summary Lists the roles a user can have.
     * @request GET:/api/roles
     * @secure
     */
    getApiRoles: (params: RequestParams = {}) =>
      this.request<RolesResp, Error>({
        path: `/api/roles`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description first, at most 20. search narrows them to games whose name holds it; sortBy "game" or "model" orders them by game name or AI platform.
     *
     * @name GetApiSessions
     * @summary Lists the caller's sessions, newest activity
     * @request GET:/api/sessions
     * @secure
     */
    getApiSessions: (
      query?: {
        search?: string;
        sortBy?: string;
      },
      params: RequestParams = {},
    ) =>
      this.request<SessionView[], Error>({
        path: `/api/sessions`,
        method: "GET",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description else's does not exist for them (404).
     *
     * @name DeleteApiSessionsId
     * @summary Deletes one of the caller's sessions; someone
     * @request DELETE:/api/sessions/{id}
     * @secure
     */
    deleteApiSessionsId: (id: string, params: RequestParams = {}) =>
      this.request<SessionDeletedResp, Error>({
        path: `/api/sessions/${id}`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Public: the login page shows it before anyone signs in. The branding as it applies, and the defaults a field the admin never set shows.
     *
     * @name GetApiSite
     * @summary Returns how the site presents itself.
     * @request GET:/api/site
     */
    getApiSite: (params: RequestParams = {}) =>
      this.request<SiteResp, Error>({
        path: `/api/site`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description Mode (development or production), uptime, last backup, active identity providers, and every registered status provider.
     *
     * @name GetApiStatus
     * @summary Service status
     * @request GET:/api/status
     */
    getApiStatus: (params: RequestParams = {}) =>
      this.request<Record<string, any>, Error>({
        path: `/api/status`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description constraint texts. Any signed-in user may read them, so organization settings can show the fallback that applies when an organization has none of its own.
     *
     * @name GetApiSystemConstraints
     * @summary Returns the three site-wide age-based prompt
     * @request GET:/api/system/constraints
     * @secure
     */
    getApiSystemConstraints: (params: RequestParams = {}) =>
      this.request<SystemConstraintsResp, Error>({
        path: `/api/system/constraints`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description the model each tier runs each task on, and the model table, each as it runs and as the code sets it, with the admin's changes marked.
     *
     * @name GetApiSystemEngineModels
     * @summary Shows the admin the engine's models: per platform
     * @request GET:/api/system/engine-models
     * @secure
     */
    getApiSystemEngineModels: (params: RequestParams = {}) =>
      this.request<EngineModelsResp, Error>({
        path: `/api/system/engine-models`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The model must be in the platform's model table and fit the task: a hearing model takes audio, an image model makes pictures, a story or speech model has a context window (400 otherwise). A picture takes a quality (low, medium, high; empty keeps the tier's). An empty model gives the task back its default. Answers the engine's models as they now are.
     *
     * @name PutApiSystemEngineModelsAssignment
     * @summary Assigns a model to a tier's task; admins only.
     * @request PUT:/api/system/engine-models/assignment
     * @secure
     */
    putApiSystemEngineModelsAssignment: (
      data: AssignEngineModelReq,
      params: RequestParams = {},
    ) =>
      this.request<EngineModelsResp, Error>({
        path: `/api/system/engine-models/assignment`,
        method: "PUT",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description the model if the table lacks it; admins only. Null facts give a code model back the code's figures and remove a model the admin added. A model a tier runs a task on cannot leave the table or lose what the task needs (400). A change to a code model applies until the next app version. Answers the engine's models as they now are.
     *
     * @name PutApiSystemEngineModelsEntry
     * @summary Sets a model's entry in the model table, adding
     * @request PUT:/api/system/engine-models/entry
     * @secure
     */
    putApiSystemEngineModelsEntry: (
      data: SetEngineModelEntryReq,
      params: RequestParams = {},
    ) =>
      this.request<EngineModelsResp, Error>({
        path: `/api/system/engine-models/entry`,
        method: "PUT",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description request its type takes; admins only. Paid by the admin's own payers on the platform (400 no_api_key without one). Text and image models only for now; any other type answers problem "no_probe". A model that does not answer is a result, with the provider's message.
     *
     * @name PostApiSystemEngineModelsProbe
     * @summary Asks whether a model answers, with the smallest
     * @request POST:/api/system/engine-models/probe
     * @secure
     */
    postApiSystemEngineModelsProbe: (
      data: ProbeEngineModelReq,
      params: RequestParams = {},
    ) =>
      this.request<ProbeEngineModelResp, Error>({
        path: `/api/system/engine-models/probe`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description models ("assignments"), or its model table back the code's figures ("entries", keeping the entries of added models a tier runs); admins only. Answers the engine's models as they now are.
     *
     * @name PostApiSystemEngineModelsReset
     * @summary Gives a platform's tasks back their default
     * @request POST:/api/system/engine-models/reset
     * @secure
     */
    postApiSystemEngineModelsReset: (
      data: ResetEngineModelsReq,
      params: RequestParams = {},
    ) =>
      this.request<EngineModelsResp, Error>({
        path: `/api/system/engine-models/reset`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiSystemSettings
     * @summary Returns the site-wide settings to an admin.
     * @request GET:/api/system/settings
     * @secure
     */
    getApiSystemSettings: (params: RequestParams = {}) =>
      this.request<SystemSettings, Error>({
        path: `/api/system/settings`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Only the fields sent change. The default AI quality tier cannot be empty; an empty free-use tier clears it; a constraint emptied, or set to its default, is the default again. The session retention is whole hours, 0 or more. Answers with the settings as they apply.
     *
     * @name PatchApiSystemSettings
     * @summary Updates the site-wide settings; admins only.
     * @request PATCH:/api/system/settings
     * @secure
     */
    patchApiSystemSettings: (
      data: UpdateSystemSettingsReq,
      params: RequestParams = {},
    ) =>
      this.request<SystemSettings, Error>({
        path: `/api/system/settings`,
        method: "PATCH",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description Fields by name: a value sets the field (an empty one hides a link or address; the site name cannot be empty), null gives it back its default. Fields not sent stay. Answers how the site now presents itself.
     *
     * @name PatchApiSystemSettingsBranding
     * @summary Changes the site's branding; admins only.
     * @request PATCH:/api/system/settings/branding
     * @secure
     */
    patchApiSystemSettingsBranding: (
      data: UpdateSiteBrandingReq,
      params: RequestParams = {},
    ) =>
      this.request<SiteResp, Error>({
        path: `/api/system/settings/branding`,
        method: "PATCH",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description key; admins only. Only a key of the admin's own; someone else's is not found. A null apiKeyId clears it.
     *
     * @name PatchApiSystemSettingsFreeUseKey
     * @summary Sets or clears the site-wide free-use API
     * @request PATCH:/api/system/settings/free-use-key
     * @secure
     */
    patchApiSystemSettingsFreeUseKey: (
      data: SetSystemFreeUseApiKeyReq,
      params: RequestParams = {},
    ) =>
      this.request<SystemSettings, Error>({
        path: `/api/system/settings/free-use-key`,
        method: "PATCH",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description The default first, for the game editor.
     *
     * @name GetApiThemes
     * @summary Lists every look the playback-engine can wear.
     * @request GET:/api/themes
     * @secure
     */
    getApiThemes: (params: RequestParams = {}) =>
      this.request<ThemeInfo[], Error>({
        path: `/api/themes`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The running version, the verified versions the app may request (with their release notes), and the last install, install request and check for new releases. Status "updater not enabled" without an updater on the run host. 403 until the app authorizes callers (app.SetUpdatesAuth).
     *
     * @name GetApiUpdates
     * @summary The instance's updates
     * @request GET:/api/updates
     * @secure
     */
    getApiUpdates: (params: RequestParams = {}) =>
      this.request<SchemafUpdateState, Error>({
        path: `/api/updates`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Asks the run host's updater to look for new releases now (fetch and verify, never install); poll GET /api/updates for the outcome. 403 until the app authorizes callers (app.SetUpdatesAuth); 409 updater_not_enabled, install_running or check_running; 429 check_too_soon (the last check finished less than a minute ago, with Retry-After).
     *
     * @name PostApiUpdatesCheck
     * @summary Check for new releases now
     * @request POST:/api/updates/check
     * @secure
     */
    postApiUpdatesCheck: (params: RequestParams = {}) =>
      this.request<void, Error>({
        path: `/api/updates/check`,
        method: "POST",
        secure: true,
        ...params,
      }),

    /**
     * @description Asks the run host's updater to install one of the available versions; the updater checks it again and decides. 403 until the app authorizes callers (app.SetUpdatesAuth); 409 updater_not_enabled or install_running; 422 version_not_available.
     *
     * @name PostApiUpdatesRequest
     * @summary Request the install of a version
     * @request POST:/api/updates/request
     * @secure
     */
    postApiUpdatesRequest: (
      data: SchemafUpdateRequest,
      params: RequestParams = {},
    ) =>
      this.request<SchemafUpdateRequest, Error>({
        path: `/api/updates/request`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description The subject of the caller.
     *
     * @name GetApiUserMe
     * @summary The authenticated caller
     * @request GET:/api/user/me
     * @secure
     */
    getApiUserMe: (params: RequestParams = {}) =>
      this.request<SchemafUser, Error>({
        path: `/api/user/me`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description total: admins every user, heads and staff the members of their organisation.
     *
     * @name GetApiUsers
     * @summary Lists a page of the users the caller may list, and the
     * @request GET:/api/users
     * @secure
     */
    getApiUsers: (
      query?: {
        filter?: UserFilter[];
        search?: string;
        sortBy?: UserSort;
        sortDir?: SortDir;
        limit?: number;
        offset?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<UserViewPage, Error>({
        path: `/api/users`,
        method: "GET",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The state is "guest" without a sign-in, "registering" for an account whose registration is incomplete, and "registered" with the account and its role. A guest whose workshop is inactive, or an account that is gone, is refused with 403. Every other endpoint refuses an account that is still registering, and a member of a suspended organisation; this one reports both.
     *
     * @name GetApiUsersMe
     * @summary Reports who is calling.
     * @request GET:/api/users/me
     * @secure
     */
    getApiUsersMe: (params: RequestParams = {}) =>
      this.request<GetCurrentUserResp, Error>({
        path: `/api/users/me`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description the caller as GET /api/users/me does. Heads, staff and individuals enter a workshop by its id; a null workshopId leaves workshop mode. A guest's place is fixed: a guest neither enters nor leaves one.
     *
     * @name PutApiUsersMeActiveWorkshop
     * @summary Enters or leaves workshop mode and answers with
     * @request PUT:/api/users/me/active-workshop
     * @secure
     */
    putApiUsersMeActiveWorkshop: (
      data: SetActiveWorkshopReq,
      params: RequestParams = {},
    ) =>
      this.request<GetCurrentUserResp, Error>({
        path: `/api/users/me/active-workshop`,
        method: "PUT",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description user: UpdateUserEndpoint with their id and only the language. The language is an ISO 639-1 code the platform supports.
     *
     * @name PatchApiUsersMeLanguage
     * @summary Sets the caller's language and answers with their
     * @request PATCH:/api/users/me/language
     * @secure
     */
    patchApiUsersMeLanguage: (
      data: UpdateMyLanguageReq,
      params: RequestParams = {},
    ) =>
      this.request<UserView, Error>({
        path: `/api/users/me/language`,
        method: "PATCH",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description with their id.
     *
     * @name GetApiUsersMeSecrets
     * @summary Is the caller's own login code: GetUserSecretsEndpoint
     * @request GET:/api/users/me/secrets
     * @secure
     */
    getApiUsersMeSecrets: (params: RequestParams = {}) =>
      this.request<UserSecretsView, Error>({
        path: `/api/users/me/secrets`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description with their id.
     *
     * @name GetApiUsersMeStats
     * @summary Is the caller's own statistics: GetUserStatsEndpoint
     * @request GET:/api/users/me/stats
     * @secure
     */
    getApiUsersMeStats: (params: RequestParams = {}) =>
      this.request<UserStatsView, Error>({
        path: `/api/users/me/stats`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The account's language follows the request's Accept-Language. Without a name it creates an account still registering, as a first Auth0 sign-in does (for testing registration).
     *
     * @name PostApiUsersNew
     * @summary Creates a user without Auth0; in dev mode only.
     * @request POST:/api/users/new
     */
    postApiUsersNew: (data: CreateUserReq, params: RequestParams = {}) =>
      this.request<User, Error>({
        path: `/api/users/new`,
        method: "POST",
        body: data,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description roles. It takes mayDelete (403 otherwise); an organisation's last head is refused (409).
     *
     * @name DeleteApiUsersId
     * @summary Deletes a user with their games, keys, shares and
     * @request DELETE:/api/users/{id}
     * @secure
     */
    deleteApiUsersId: (id: string, params: RequestParams = {}) =>
      this.request<void, Error>({
        path: `/api/users/${id}`,
        method: "DELETE",
        secure: true,
        ...params,
      }),

    /**
     * @description heads and staff their organisation's members and the guests in its workshops; admins anyone. 404 otherwise.
     *
     * @name GetApiUsersId
     * @summary Returns a user the caller may see: themselves, and for
     * @request GET:/api/users/{id}
     * @secure
     */
    getApiUsersId: (id: string, params: RequestParams = {}) =>
      this.request<UserView, Error>({
        path: `/api/users/${id}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Send what changes; a field left out stays as it is. Each changed section must be one the caller may edit (mayEditCard for the name, mayEditAccount for language, age group, AI quality tier and default key, mayEditContact for the email); otherwise the whole request is refused (403). The age group is "u13" (13-17, no parental consent), "u13p" (13-17, with parental consent) or "u18" (18+). An empty aiQualityTier or email removes it.
     *
     * @name PostApiUsersId
     * @summary Changes a user and answers with them as stored.
     * @request POST:/api/users/{id}
     * @secure
     */
    postApiUsersId: (
      id: string,
      data: UpdateUserReq,
      params: RequestParams = {},
    ) =>
      this.request<UserView, Error>({
        path: `/api/users/${id}`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description an individual of no organisation. A head removes a member from the organisation instead (DELETE /api/institutions/{id}/members/{userID}).
     *
     * @name DeleteApiUsersIdRole
     * @summary Takes a user's role away, the admin's: they become
     * @request DELETE:/api/users/{id}/role
     * @secure
     */
    deleteApiUsersIdRole: (id: string, params: RequestParams = {}) =>
      this.request<UserView, Error>({
        path: `/api/users/${id}/role`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The admin gives any role, scoped to an organisation or a workshop by their ids; a head makes a member of their organisation head or staff there. Only an individual becomes admin, and an organisation keeps its last head (409 last_head).
     *
     * @name PostApiUsersIdRole
     * @summary Gives a user a new role and answers with them.
     * @request POST:/api/users/{id}/role
     * @secure
     */
    postApiUsersIdRole: (
      id: string,
      data: SetUserRoleReq,
      params: RequestParams = {},
    ) =>
      this.request<UserView, Error>({
        path: `/api/users/${id}/role`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description for heads and staff the guests' in their organisation's workshops, to help a guest who lost theirs. Both fields are null for an account without one.
     *
     * @name GetApiUsersIdSecrets
     * @summary Returns a user's login code: the guest's own, and
     * @request GET:/api/users/{id}/secrets
     * @secure
     */
    getApiUsersIdSecrets: (id: string, params: RequestParams = {}) =>
      this.request<UserSecretsView, Error>({
        path: `/api/users/${id}/secrets`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description takes seeing their account (403 otherwise).
     *
     * @name GetApiUsersIdStats
     * @summary Returns what a user did on the platform, counted. It
     * @request GET:/api/users/{id}/stats
     * @secure
     */
    getApiUsersIdStats: (id: string, params: RequestParams = {}) =>
      this.request<UserStatsView, Error>({
        path: `/api/users/${id}/stats`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description may see, waiting first: every one for an admin, the allow list.
     *
     * @name GetApiVerifiedPages
     * @summary Lists a page of the verified pages the caller
     * @request GET:/api/verified-pages
     * @secure
     */
    getApiVerifiedPages: (
      query?: {
        filter?: VerifiedPageFilter[];
        search?: string;
        sortBy?: VerifiedPageSort;
        sortDir?: SortDir;
        limit?: number;
        offset?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<VerifiedPageViewPage, Error>({
        path: `/api/verified-pages`,
        method: "GET",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description for it start nothing until they get another.
     *
     * @name DeleteApiVerifiedPagesId
     * @summary Removes a verified page; the games embedded
     * @request DELETE:/api/verified-pages/{id}
     * @secure
     */
    deleteApiVerifiedPagesId: (id: string, params: RequestParams = {}) =>
      this.request<MessageResp, Error>({
        path: `/api/verified-pages/${id}`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description callback; its page ID stays.
     *
     * @name PatchApiVerifiedPagesId
     * @summary Changes a verified page's address and
     * @request PATCH:/api/verified-pages/{id}
     * @secure
     */
    patchApiVerifiedPagesId: (
      id: string,
      data: UpdateVerifiedPageReq,
      params: RequestParams = {},
    ) =>
      this.request<VerifiedPageView, Error>({
        path: `/api/verified-pages/${id}`,
        method: "PATCH",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description approval, for an admin.
     *
     * @name PutApiVerifiedPagesIdApproval
     * @summary Approves a verified page or withdraws the
     * @request PUT:/api/verified-pages/{id}/approval
     * @secure
     */
    putApiVerifiedPagesIdApproval: (
      id: string,
      data: ApproveVerifiedPageReq,
      params: RequestParams = {},
    ) =>
      this.request<VerifiedPageView, Error>({
        path: `/api/verified-pages/${id}/approval`,
        method: "PUT",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description Returns the version of the running release ("dev" where none runs, as in development). Frontend and server ship together, so a page compares it with the version it was loaded with to see that the server was upgraded.
     *
     * @name GetApiVersion
     * @summary Reports the release the server runs.
     * @request GET:/api/version
     */
    getApiVersion: (params: RequestParams = {}) =>
      this.request<GetVersionResp, Error>({
        path: `/api/version`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description answers its sample, as VoiceSampleEndpoint does. The voice made before for that key and style, and its sample, are deleted: every game with exactly this voice style on this key speaks in the new voice from now on. Only for a voice set to custom, as VoiceSampleEndpoint; an empty voice style makes none and only plays the built-in voice.
     *
     * @name PostApiVoicesNew
     * @summary Makes a new voice for a voice style on the paying key and
     * @request POST:/api/voices/new
     * @secure
     */
    postApiVoicesNew: (data: VoiceSampleReq, params: RequestParams = {}) =>
      this.request<VoiceSampleResp, Error>({
        path: `/api/voices/new`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description neutral sample line, the same for every game, in the caller's language, and answers it as audio. It reads the form as it is, so a game not yet saved is heard too. Only a voice set to custom, in a genre that offers it, is heard (a choice in voiceSlots of /api/genres): with a built-in choice the session casts its own voice, so a sample would mislead, and anything else is refused with 400 voice_not_custom. The voice made for the voice style on the paying key speaks: the first sample makes the voice when there is none and keeps the sample, and later ones play it without asking the provider (stored: true). With an empty voice style a built-in voice speaks. The caller pays with their own keys, as in the editor: workshop, organisation, own, site; never a game's sponsor. None that speaks live: 400 no_api_key; a key its provider rejects: 400 invalid_api_key; a provider that fails: 503 ai_error.
     *
     * @name PostApiVoicesSample
     * @summary Has the voice made for a game's voice style say a
     * @request POST:/api/voices/sample
     * @secure
     */
    postApiVoicesSample: (data: VoiceSampleReq, params: RequestParams = {}) =>
      this.request<VoiceSampleResp, Error>({
        path: `/api/voices/sample`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description and the total: an admin all of them, a head or staff their organisation's. institutionId narrows it to one organisation.
     *
     * @name GetApiWorkshops
     * @summary Lists a page of the workshops the caller may list,
     * @request GET:/api/workshops
     * @secure
     */
    getApiWorkshops: (
      query?: {
        institutionId?: string;
        filter?: WorkshopFilter[];
        search?: string;
        sortBy?: WorkshopSort;
        sortDir?: SortDir;
        limit?: number;
        offset?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<WorkshopViewPage, Error>({
        path: `/api/workshops`,
        method: "GET",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description for the admin, in the one named (institutionId).
     *
     * @name PostApiWorkshops
     * @summary Creates a workshop in the caller's organisation, or,
     * @request POST:/api/workshops
     * @secure
     */
    postApiWorkshops: (data: CreateWorkshopReq, params: RequestParams = {}) =>
      this.request<WorkshopView, Error>({
        path: `/api/workshops`,
        method: "POST",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description caller's organisation's workshops: for its head and staff, helping a guest who lost it.
     *
     * @name GetApiWorkshopsGuestsGuestIdToken
     * @summary Returns the login code of a guest in one of the
     * @request GET:/api/workshops/guests/{guestId}/token
     * @secure
     */
    getApiWorkshopsGuestsGuestIdToken: (
      guestId: string,
      params: RequestParams = {},
    ) =>
      this.request<GuestTokenResp, Error>({
        path: `/api/workshops/guests/${guestId}/token`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description workshops a new login code: for its head and staff. Earlier login links stop working and the guest's sessions end.
     *
     * @name PostApiWorkshopsGuestsGuestIdTokenReset
     * @summary Gives a guest in one of the caller's organisation's
     * @request POST:/api/workshops/guests/{guestId}/token/reset
     * @secure
     */
    postApiWorkshopsGuestsGuestIdTokenReset: (
      guestId: string,
      params: RequestParams = {},
    ) =>
      this.request<GuestTokenResp, Error>({
        path: `/api/workshops/guests/${guestId}/token/reset`,
        method: "POST",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description stay with their creators.
     *
     * @name DeleteApiWorkshopsId
     * @summary Deletes a workshop with its guests; its other games
     * @request DELETE:/api/workshops/{id}
     * @secure
     */
    deleteApiWorkshopsId: (id: string, params: RequestParams = {}) =>
      this.request<MessageResp, Error>({
        path: `/api/workshops/${id}`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description organisation's, the one they are in, one with a public page, or any, for an admin. Others get 404.
     *
     * @name GetApiWorkshopsId
     * @summary Returns a workshop the caller may read: one of their
     * @request GET:/api/workshops/{id}
     * @secure
     */
    getApiWorkshopsId: (id: string, params: RequestParams = {}) =>
      this.request<WorkshopView, Error>({
        path: `/api/workshops/${id}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Only what is given changes. An empty aiQualityTier, promptConstraints or publicDescription removes it; an empty publicLinks list clears the links. Switching public off ends the public page's share links.
     *
     * @name PatchApiWorkshopsId
     * @summary Changes a workshop and answers with it as stored.
     * @request PATCH:/api/workshops/{id}
     * @secure
     */
    patchApiWorkshopsId: (
      id: string,
      data: UpdateWorkshopReq,
      params: RequestParams = {},
    ) =>
      this.request<WorkshopView, Error>({
        path: `/api/workshops/${id}`,
        method: "PATCH",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description keys given to its organisation; null or empty clears it, and the workshop plays no more until one is picked.
     *
     * @name PutApiWorkshopsIdApiKey
     * @summary Sets the key a workshop plays with: one of the
     * @request PUT:/api/workshops/{id}/api-key
     * @secure
     */
    putApiWorkshopsIdApiKey: (
      id: string,
      data: SetWorkshopApiKeyReq,
      params: RequestParams = {},
    ) =>
      this.request<WorkshopView, Error>({
        path: `/api/workshops/${id}/api-key`,
        method: "PUT",
        body: data,
        secure: true,
        type: "application/json",
        format: "json",
        ...params,
      }),

    /**
     * @description The stream opens with a "connected" event carrying the workshop id, then sends each update, until the client disconnects.
     *
     * @name GetApiWorkshopsIdEvents
     * @summary Streams a workshop's updates as Server-Sent Events.
     * @request GET:/api/workshops/{id}/events
     * @secure
     */
    getApiWorkshopsIdEvents: (id: string, params: RequestParams = {}) =>
      this.request<WorkshopEvent, Error>({
        path: `/api/workshops/${id}/events`,
        method: "GET",
        secure: true,
        ...params,
      }),

    /**
     * @description loses their place, a member working in it leaves it.
     *
     * @name DeleteApiWorkshopsIdMembersUserId
     * @summary Takes someone out of a workshop: a guest of it
     * @request DELETE:/api/workshops/{id}/members/{userId}
     * @secure
     */
    deleteApiWorkshopsIdMembersUserId: (
      id: string,
      userId: string,
      params: RequestParams = {},
    ) =>
      this.request<MessageResp, Error>({
        path: `/api/workshops/${id}/members/${userId}`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description the total: for those who may list them (an empty list for others); a workshop the caller may not read, 404.
     *
     * @name GetApiWorkshopsIdParticipants
     * @summary Lists a page of a workshop's participants, and
     * @request GET:/api/workshops/{id}/participants
     * @secure
     */
    getApiWorkshopsIdParticipants: (
      id: string,
      query?: {
        filter?: string[];
        search?: string;
        sortBy?: ParticipantSort;
        sortDir?: SortDir;
        limit?: number;
        offset?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<WorkshopParticipantViewPage, Error>({
        path: `/api/workshops/${id}/participants`,
        method: "GET",
        query: query,
        secure: true,
        format: "json",
        ...params,
      }),
  };
}
