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

export interface AcceptInviteResp {
  authToken?: string;
  message: string;
  user?: User;
}

export interface AddWorkshopMemberReq {
  userId: string;
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
  linkedShares: ApiKeyShare[];
  share?: ApiKeyShare;
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

export interface ApiKeysResp {
  apiKeys: ApiKey[];
  shares: ApiKeyShare[];
}

export interface AvailableKey {
  isDefault: boolean;
  name: string;
  platform: string;
  /** @format uuid */
  shareId: string;
  source: string;
}

export interface CheckNameResp {
  available: boolean;
  profane?: boolean;
}

export interface Choice {
  tier: string;
  toggles: Toggles;
}

export interface ChooseRequest {
  choice?: Choice;
  offer: string;
}

export interface ConnectRequest {
  sdp: string;
}

export interface ConnectResponse {
  sdp: string;
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
  description?: string;
  genre?: string;
  name: string;
  public?: boolean;
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
  name: string;
}

export interface CreateRequest {
  game?: string;
  language?: string;
  once: string;
  share?: string;
}

export interface CreateResponse {
  id: string;
}

export interface CreateUserReq {
  email?: string;
  name: string;
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

export interface EnrichedGameShare {
  aiQualityTier?: string;
  /** @format uuid */
  apiKeyShareId: string;
  /** @format date-time */
  createdAt: string;
  /** @format uuid */
  createdBy?: string;
  /** @format uuid */
  gameId: string;
  gameName?: string;
  /** @format uuid */
  id: string;
  /** @format uuid */
  institutionId?: string;
  remaining?: number;
  shareUrl: string;
  source: string;
  token: string;
  /** @format uuid */
  workshopId?: string;
  workshopName?: string;
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
  /** @format uuid */
  workshopId?: string;
}

export interface GameApiKeyStatusResp {
  available: boolean;
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
}

export interface GameSession {
  /** @format uuid */
  apiKeyShareId?: string;
  /** @format date-time */
  createdAt: string;
  /** @format uuid */
  gameId: string;
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

export interface GameShareResp {
  aiQualityTier?: string;
  /** @format uuid */
  apiKeyShareId: string;
  /** @format date-time */
  createdAt: string;
  /** @format uuid */
  createdBy?: string;
  /** @format uuid */
  gameId: string;
  /** @format uuid */
  id: string;
  /** @format uuid */
  institutionId?: string;
  remaining?: number;
  shareUrl: string;
  token: string;
  /** @format uuid */
  workshopId?: string;
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

export interface GenreInfo {
  id: string;
  needs: string[];
}

export interface GetCurrentUserResp {
  state: string;
  user?: User;
}

export interface GetVersionResp {
  buildTime: string;
  gitCommit: string;
  version: string;
}

export interface GraphResponse {
  graph: string;
}

export interface GuestGameInfoResp {
  description?: string;
  /** @format uuid */
  gameId: string;
  name: string;
  public: boolean;
  remaining?: number;
}

export interface InputRequest {
  /** @format byte */
  audio?: Blob;
  text?: string;
}

export interface Institution {
  freeUseAiQualityTier?: string;
  /** @format uuid */
  freeUseApiKeyShareId?: string;
  /** @format uuid */
  id: string;
  members?: InstitutionMember[];
  meta: Meta;
  name: string;
  promptConstraints?: string;
}

export interface InstitutionMember {
  email?: string;
  name: string;
  role: string;
  /** @format uuid */
  userId: string;
}

export interface InviteResp {
  createdAt: string;
  expiresAt?: string;
  id: string;
  institutionId: string;
  institutionName?: string;
  inviteToken?: string;
  invitedEmail?: string;
  invitedUserId?: string;
  maxUses?: number;
  modifiedAt?: string;
  role: string;
  status: string;
  usesCount: number;
  workshopId?: string;
  workshopName?: string;
}

export interface Language {
  iso: string;
  label: string;
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

export interface NodeDetail {
  inputs: Sample[];
  name: string;
  outputs: Sample[];
  phase: string;
  role: string;
  type: string;
  usage?: UsageRecord;
}

export interface OfferOption {
  customizable: boolean;
  default: Choice;
  name: string;
  offer: string;
  platform: string;
  tiers: TierOption[];
}

export interface Options {
  choosing: boolean;
  offers: OfferOption[];
  settled?: Settled;
}

export interface ParticipantLoginReq {
  token: string;
}

export interface ParticipantLoginResp {
  token: string;
  /** @format uuid */
  userId: string;
}

export interface PlatformInfo {
  genres: string[];
  id: string;
  name: string;
  tiers: TierModels[];
  toggles: Toggles;
  userKeys: boolean;
}

export interface Preference {
  offer?: string;
  platform: string;
  tier: string;
  toggles: Toggles;
}

export interface PrivateShareStatusResp {
  shares: EnrichedGameShare[];
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

export interface Registration {
  ageGroup: string;
  email?: string;
  language?: string;
  name: string;
}

export interface RolesResp {
  roles: string[];
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

export interface SetActiveWorkshopReq {
  /** @format uuid */
  workshopId?: string;
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
  platform: string;
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

export interface Snapshot {
  choosing: boolean;
  conversation?: string;
  phases: Record<string, string>;
  props: Record<string, string>;
  settled?: Settled;
  started: boolean;
  usage: UsageReport;
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
  promptConstraintU13?: string;
  promptConstraintU13p?: string;
  promptConstraintU18?: string;
}

export interface ThemeInfo {
  id: string;
}

export interface TierModels {
  id: string;
  label: string;
  models: Record<string, Model>;
}

export interface TierOption {
  label: string;
  models: Record<string, string>;
  tier: string;
  toggles: Toggles;
}

export interface ToggleRequest {
  on: boolean;
  toggle: string;
}

export interface Toggles {
  "audio-in": boolean;
  "audio-live": boolean;
  "audio-out": boolean;
  images: boolean;
}

export interface Topology {
  edges: TopologyEdge[];
  imagery: string;
  inputs: string[];
  language?: string;
  name: string;
  nodes: TopologyNode[];
  theme?: string;
  title?: string;
}

export interface TopologyEdge {
  from: string;
  kind: string;
  to: string;
}

export interface TopologyNode {
  name: string;
  role: string;
}

export interface UpdateApiKeyReq {
  name?: string;
}

export interface UpdateGameReq {
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
  /** @format uuid */
  workshopId?: string;
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
}

export interface UpdateInstitutionPromptConstraintsReq {
  promptConstraints?: string;
}

export interface UpdateInstitutionReq {
  name: string;
}

export interface UpdateSystemSettingsReq {
  defaultAiQualityTier?: string;
  freeUseAiQualityTier?: string;
  promptConstraintU13?: string;
  promptConstraintU13p?: string;
  promptConstraintU18?: string;
}

export interface UpdateUserLanguageReq {
  language: string;
}

export interface UpdateUserReq {
  ageGroup?: string;
  aiQualityTier?: string;
  /** @format uuid */
  defaultApiKeyShareId?: string;
  email?: string;
  name?: string;
}

export interface UpdateWorkshopReq {
  active: boolean;
  aiQualityTier?: string;
  allowGameSharing: boolean;
  isPaused: boolean;
  name: string;
  promptConstraints?: string;
  public: boolean;
  publicDescription?: string;
  publicLinks?: PublicWorkshopLink[];
  publicSlug?: string;
  showOtherParticipantsGames: boolean;
  showPublicGames: boolean;
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

export interface UserParticipantCodeResp {
  token?: string;
  words?: string;
}

export interface UserRole {
  /** @format uuid */
  id: string;
  institution?: Institution;
  meta: Meta;
  role: string;
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
  role: string;
  status: string;
  usesCount: number;
  /** @format uuid */
  workshopId?: string;
}

export interface UserSessionWithGame {
  /** @format uuid */
  apiKeyShareId?: string;
  /** @format date-time */
  createdAt: string;
  /** @format uuid */
  gameId: string;
  gameName: string;
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

export interface UserStats {
  gamesCreated: number;
  gamesPlayed: number;
  messagesSent: number;
  totalPlaysOnGames: number;
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
  type: string;
  /** @format uuid */
  workshopId?: string;
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
  role: string;
  /** @format uuid */
  workshopId: string;
}

export interface WorkshopParticipantTokenResp {
  token: string;
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

export enum ContentType {
  Json = "application/json",
  JsonApi = "application/vnd.api+json",
  FormData = "multipart/form-data",
  UrlEncoded = "application/x-www-form-urlencoded",
  Text = "text/plain",
}

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
    [ContentType.Json]: (input: any) =>
      input !== null && (typeof input === "object" || typeof input === "string")
        ? JSON.stringify(input)
        : input,
    [ContentType.JsonApi]: (input: any) =>
      input !== null && (typeof input === "object" || typeof input === "string")
        ? JSON.stringify(input)
        : input,
    [ContentType.Text]: (input: any) =>
      input !== null && typeof input !== "string"
        ? JSON.stringify(input)
        : input,
    [ContentType.FormData]: (input: any) => {
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
    [ContentType.UrlEncoded]: (input: any) => this.toQueryString(input),
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
    const payloadFormatter = this.contentFormatters[type || ContentType.Json];
    const responseFormat = format || requestParams.format;

    return this.customFetch(
      `${baseUrl || this.baseUrl || ""}${path}${queryString ? `?${queryString}` : ""}`,
      {
        ...requestParams,
        headers: {
          ...(requestParams.headers || {}),
          ...(type && type !== ContentType.FormData
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
     * @description (organization shares, sponsorships).
     *
     * @name GetApiApikeys
     * @summary Lists the caller's API keys and all their linked shares
     * @request GET:/api/apikeys
     * @secure
     */
    getApiApikeys: (params: RequestParams = {}) =>
      this.request<ApiKeysResp, Error>({
        path: `/api/apikeys`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Platform and key are required; surrounding whitespace is trimmed. Without a name the key is named after its platform and today's date, e.g. "Mistral 12.02.26".
     *
     * @name PostApiApikeysNew
     * @summary Creates an API key and shares it with the caller.
     * @request POST:/api/apikeys/new
     * @secure
     */
    postApiApikeysNew: (data: CreateApiKeyReq, params: RequestParams = {}) =>
      this.request<ApiKeyShare, Error>({
        path: `/api/apikeys/new`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description With cascade=true it deletes the key and all its shares.
     *
     * @name DeleteApiApikeysId
     * @summary Removes an API key share and answers with it as it was.
     * @request DELETE:/api/apikeys/{id}
     * @secure
     */
    deleteApiApikeysId: (id: string, params: RequestParams = {}) =>
      this.request<ApiKeyShare, Error>({
        path: `/api/apikeys/${id}`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiApikeysId
     * @summary Returns an API key share and the shares linked to it.
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
     * @summary Renames an API key.
     * @request PATCH:/api/apikeys/{id}
     * @secure
     */
    patchApiApikeysId: (
      id: string,
      data: UpdateApiKeyReq,
      params: RequestParams = {},
    ) =>
      this.request<ApiKeyShare, Error>({
        path: `/api/apikeys/${id}`,
        method: "PATCH",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description institution, at the path older clients use.
     *
     * @name PostApiApikeysId
     * @summary Shares an API key with a user, workshop or
     * @request POST:/api/apikeys/{id}
     * @secure
     */
    postApiApikeysId: (
      id: string,
      data: ShareApiKeyReq,
      params: RequestParams = {},
    ) =>
      this.request<ApiKeyShare, Error>({
        path: `/api/apikeys/${id}`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name PutApiApikeysIdDefault
     * @summary Makes an API key the caller's default for new sessions.
     * @request PUT:/api/apikeys/{id}/default
     * @secure
     */
    putApiApikeysIdDefault: (id: string, params: RequestParams = {}) =>
      this.request<ApiKeyShare, Error>({
        path: `/api/apikeys/${id}/default`,
        method: "PUT",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description with each game's name and where the share comes from. Query: context=personal lists all shares and requires owning the key; context=organization lists only organization and workshop shares.
     *
     * @name GetApiApikeysIdGameShares
     * @summary Lists the game share links that use an API key,
     * @request GET:/api/apikeys/{id}/game-shares
     * @secure
     */
    getApiApikeysIdGameShares: (id: string, params: RequestParams = {}) =>
      this.request<EnrichedGameShare[], Error>({
        path: `/api/apikeys/${id}/game-shares`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name PostApiApikeysIdShares
     * @summary Shares an API key with a user, workshop or institution.
     * @request POST:/api/apikeys/{id}/shares
     * @secure
     */
    postApiApikeysIdShares: (
      id: string,
      data: ShareApiKeyReq,
      params: RequestParams = {},
    ) =>
      this.request<ApiKeyShare, Error>({
        path: `/api/apikeys/${id}/shares`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
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
        type: ContentType.Json,
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
    getApiAuthCheckName: (params: RequestParams = {}) =>
      this.request<CheckNameResp, Error>({
        path: `/api/auth/check-name`,
        method: "GET",
        format: "json",
        ...params,
      }),

    /**
     * @description Development only: outside development mode it answers 404. Takes a preset role ("admin-1", "admin-2", "head-1", "head-2", "staff-1", "staff-2", "individual-1", "individual-2", "participant"), starts a session for its account and sets the session cookie.
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
        type: ContentType.Json,
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
     * @description Takes the participant token whole, or its words from a link, QR code or typed code. Starts a session, sets the session cookie and answers with the session token, for Bearer clients, and the participant's id.
     *
     * @name PostApiAuthParticipantLogin
     * @summary Signs a workshop participant in with their token.
     * @request POST:/api/auth/participant-login
     */
    postApiAuthParticipantLogin: (
      data: ParticipantLoginReq,
      params: RequestParams = {},
    ) =>
      this.request<ParticipantLoginResp, Error>({
        path: `/api/auth/participant-login`,
        method: "POST",
        body: data,
        type: ContentType.Json,
        format: "json",
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
     * @description An account exists from its first Auth0 sign-in and is unusable until it has a name and an age group; the email is optional. An account registers once.
     *
     * @name PostApiAuthRegister
     * @summary Completes the caller's registration.
     * @request POST:/api/auth/register
     * @secure
     */
    postApiAuthRegister: (data: Registration, params: RequestParams = {}) =>
      this.request<User, Error>({
        path: `/api/auth/register`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
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
        type: ContentType.Json,
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
        type: ContentType.Json,
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
        type: ContentType.Json,
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
        type: ContentType.Json,
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
        type: ContentType.Json,
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
        type: ContentType.Json,
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
        type: ContentType.Json,
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
     * @description Query: search (name, case-insensitive), sortBy (name, createdAt, modifiedAt, playCount), sortDir (asc, desc), filter (all, own, public, organization, favorites).
     *
     * @name GetApiGames
     * @summary Lists the games the caller can see.
     * @request GET:/api/games
     * @secure
     */
    getApiGames: (params: RequestParams = {}) =>
      this.request<Game[], Error>({
        path: `/api/games`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiGamesFavourites
     * @summary Lists the games the caller has marked as favourites.
     * @request GET:/api/games/favourites
     * @secure
     */
    getApiGamesFavourites: (params: RequestParams = {}) =>
      this.request<Game[], Error>({
        path: `/api/games/favourites`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Only the name is required, at most 70 characters; the genre is one of the engine's genres and makes an adventure when empty.
     *
     * @name PostApiGamesNew
     * @summary Creates a game owned by the caller.
     * @request POST:/api/games/new
     * @secure
     */
    postApiGamesNew: (data: CreateGameReq, params: RequestParams = {}) =>
      this.request<Game, Error>({
        path: `/api/games/new`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name DeleteApiGamesId
     * @summary Deletes a game and answers with it as it was.
     * @request DELETE:/api/games/{id}
     * @secure
     */
    deleteApiGamesId: (id: string, params: RequestParams = {}) =>
      this.request<Game, Error>({
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
      this.request<Game, Error>({
        path: `/api/games/${id}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description Takes the whole game; its name is at most 70 characters. Answers with the game as stored.
     *
     * @name PostApiGamesId
     * @summary Updates a game's properties.
     * @request POST:/api/games/{id}
     * @secure
     */
    postApiGamesId: (
      id: string,
      data: UpdateGameReq,
      params: RequestParams = {},
    ) =>
      this.request<Game, Error>({
        path: `/api/games/${id}`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description caller playing a game.
     *
     * @name GetApiGamesIdApiKeyStatus
     * @summary Answers whether an API key resolves for the
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
     * @description with, in order of priority.
     *
     * @name GetApiGamesIdAvailableKeys
     * @summary Lists the API keys the caller can play a game
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
     * @description new private game of the caller's.
     *
     * @name PostApiGamesIdClone
     * @summary Copies a public game, or one of the caller's own, into a
     * @request POST:/api/games/{id}/clone
     * @secure
     */
    postApiGamesIdClone: (id: string, params: RequestParams = {}) =>
      this.request<Game, Error>({
        path: `/api/games/${id}/clone`,
        method: "POST",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name DeleteApiGamesIdFavourite
     * @summary Removes a game from the caller's favourites.
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
     * No description
     *
     * @name PostApiGamesIdFavourite
     * @summary Adds a game to the caller's favourites.
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
     * @description has access to. These are the caller's own personal and organisation shares, their institution's shares for head, staff and admins, the shares of the workshop they are in, and for head and staff outside a workshop the shares of every workshop of their institution.
     *
     * @name GetApiGamesIdPrivateShare
     * @summary Lists the share links of a game the caller
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
     * No description
     *
     * @name GetApiGamesIdSessions
     * @summary Lists the caller's own sessions of a game.
     * @request GET:/api/games/{id}/sessions
     * @secure
     */
    getApiGamesIdSessions: (id: string, params: RequestParams = {}) =>
      this.request<GameSession[], Error>({
        path: `/api/games/${id}/sessions`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description A workshop share (workshopId set) plays on the workshop's default API key and is reused when the game already has one in that workshop. A personal or organisation share needs sponsorKeyShareId; it is an organisation share when that key share belongs to an institution.
     *
     * @name PostApiGamesIdShares
     * @summary Creates a share link for a game.
     * @request POST:/api/games/{id}/shares
     * @secure
     */
    postApiGamesIdShares: (
      id: string,
      data: CreateGameShareReq,
      params: RequestParams = {},
    ) =>
      this.request<GameShareResp, Error>({
        path: `/api/games/${id}/shares`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description The share's creator, the game's owner, members of the share's workshop and head or staff of its institution may revoke it.
     *
     * @name DeleteApiGamesIdSharesShareId
     * @summary Revokes a share link and cleans up its guest data.
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
     * @description Whoever may revoke the share may update it.
     *
     * @name PatchApiGamesIdSharesShareId
     * @summary Updates a share link's settings.
     * @request PATCH:/api/games/{id}/shares/{shareId}
     * @secure
     */
    patchApiGamesIdSharesShareId: (
      id: string,
      shareId: string,
      data: UpdateGameShareReq,
      params: RequestParams = {},
    ) =>
      this.request<GameShareResp, Error>({
        path: `/api/games/${id}/shares/${shareId}`,
        method: "PATCH",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description answers with the game.
     *
     * @name DeleteApiGamesIdSponsor
     * @summary Removes the public sponsorship from a game and
     * @request DELETE:/api/games/{id}/sponsor
     * @secure
     */
    deleteApiGamesIdSponsor: (id: string, params: RequestParams = {}) =>
      this.request<Game, Error>({
        path: `/api/games/${id}/sponsor`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description API key shares and answers with the game.
     *
     * @name PutApiGamesIdSponsor
     * @summary Sponsors a game publicly with one of the caller's
     * @request PUT:/api/games/{id}/sponsor
     * @secure
     */
    putApiGamesIdSponsor: (
      id: string,
      data: GameSponsorReq,
      params: RequestParams = {},
    ) =>
      this.request<Game, Error>({
        path: `/api/games/${id}/sponsor`,
        method: "PUT",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiGamesIdYaml
     * @summary Exports a game as a game file, YAML by default.
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
     * @description answers with the game. The file's content replaces the game's; its id is ignored.
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
      this.request<Game, Error>({
        path: `/api/games/${id}/yaml`,
        method: "PUT",
        body: data,
        secure: true,
        type: ContentType.Json,
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
     * No description
     *
     * @name GetApiInstitutions
     * @summary Lists the institutions the caller can see.
     * @request GET:/api/institutions
     * @secure
     */
    getApiInstitutions: (params: RequestParams = {}) =>
      this.request<Institution[], Error>({
        path: `/api/institutions`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The name is required.
     *
     * @name PostApiInstitutions
     * @summary Creates an institution.
     * @request POST:/api/institutions
     * @secure
     */
    postApiInstitutions: (
      data: CreateInstitutionReq,
      params: RequestParams = {},
    ) =>
      this.request<Institution, Error>({
        path: `/api/institutions`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name DeleteApiInstitutionsId
     * @summary Deletes an institution; admins only.
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
     * No description
     *
     * @name GetApiInstitutionsId
     * @summary Returns an institution.
     * @request GET:/api/institutions/{id}
     * @secure
     */
    getApiInstitutionsId: (id: string, params: RequestParams = {}) =>
      this.request<Institution, Error>({
        path: `/api/institutions/${id}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The name is required.
     *
     * @name PatchApiInstitutionsId
     * @summary Renames an institution.
     * @request PATCH:/api/institutions/{id}
     * @secure
     */
    patchApiInstitutionsId: (
      id: string,
      data: UpdateInstitutionReq,
      params: RequestParams = {},
    ) =>
      this.request<Institution, Error>({
        path: `/api/institutions/${id}`,
        method: "PATCH",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description heads and staff only.
     *
     * @name GetApiInstitutionsIdApikeys
     * @summary Lists the API keys shared with an institution;
     * @request GET:/api/institutions/{id}/apikeys
     * @secure
     */
    getApiInstitutionsIdApikeys: (id: string, params: RequestParams = {}) =>
      this.request<ApiKeyShare[], Error>({
        path: `/api/institutions/${id}/apikeys`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description free-use API key share and answers with the institution. Any member of the institution plays for free with this key. A null shareId clears it.
     *
     * @name PatchApiInstitutionsIdFreeUseKey
     * @summary Sets or clears an institution's
     * @request PATCH:/api/institutions/{id}/free-use-key
     * @secure
     */
    patchApiInstitutionsIdFreeUseKey: (
      id: string,
      data: SetInstitutionFreeUseKeyReq,
      params: RequestParams = {},
    ) =>
      this.request<Institution, Error>({
        path: `/api/institutions/${id}/free-use-key`,
        method: "PATCH",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiInstitutionsIdMembers
     * @summary Lists the members of an institution.
     * @request GET:/api/institutions/{id}/members
     * @secure
     */
    getApiInstitutionsIdMembers: (id: string, params: RequestParams = {}) =>
      this.request<User[], Error>({
        path: `/api/institutions/${id}/members`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description and admins only.
     *
     * @name DeleteApiInstitutionsIdMembersUserId
     * @summary Removes a member from an institution; heads
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
     * @description prompt constraints and answers with the institution; heads and admins only. An empty or null promptConstraints clears them.
     *
     * @name PatchApiInstitutionsIdPromptConstraints
     * @summary Sets or clears an institution's
     * @request PATCH:/api/institutions/{id}/prompt-constraints
     * @secure
     */
    patchApiInstitutionsIdPromptConstraints: (
      id: string,
      data: UpdateInstitutionPromptConstraintsReq,
      params: RequestParams = {},
    ) =>
      this.request<Institution, Error>({
        path: `/api/institutions/${id}/prompt-constraints`,
        method: "PATCH",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiInvites
     * @summary Lists the invites addressed to the caller.
     * @request GET:/api/invites
     * @secure
     */
    getApiInvites: (params: RequestParams = {}) =>
      this.request<InviteResp[], Error>({
        path: `/api/invites`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiInvitesAll
     * @summary Lists every invite; admins only.
     * @request GET:/api/invites/all
     * @secure
     */
    getApiInvitesAll: (params: RequestParams = {}) =>
      this.request<InviteResp[], Error>({
        path: `/api/invites/all`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description head, staff or individual. The role defaults to staff. The invite names the invited user by id or email.
     *
     * @name PostApiInvitesInstitution
     * @summary Invites a user to join an institution as
     * @request POST:/api/invites/institution
     * @secure
     */
    postApiInvitesInstitution: (
      data: CreateInstitutionInviteReq,
      params: RequestParams = {},
    ) =>
      this.request<UserRoleInvite, Error>({
        path: `/api/invites/institution`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description The caller is head or staff of the institution, or an admin.
     *
     * @name GetApiInvitesInstitutionInstitutionId
     * @summary Lists the invites to join an institution.
     * @request GET:/api/invites/institution/{institutionId}
     * @secure
     */
    getApiInvitesInstitutionInstitutionId: (
      institutionId: string,
      params: RequestParams = {},
    ) =>
      this.request<InviteResp[], Error>({
        path: `/api/invites/institution/${institutionId}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description workshop as participants.
     *
     * @name PostApiInvitesWorkshop
     * @summary Creates an open invite for users to join a
     * @request POST:/api/invites/workshop
     * @secure
     */
    postApiInvitesWorkshop: (
      data: CreateWorkshopInviteReq,
      params: RequestParams = {},
    ) =>
      this.request<UserRoleInvite, Error>({
        path: `/api/invites/workshop`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description join a workshop. A second pending invite for the same user and workshop is refused with 409.
     *
     * @name PostApiInvitesWorkshopEmail
     * @summary Invites a registered user, by email, to
     * @request POST:/api/invites/workshop/email
     * @secure
     */
    postApiInvitesWorkshopEmail: (
      data: CreateWorkshopEmailInviteReq,
      params: RequestParams = {},
    ) =>
      this.request<InviteResp, Error>({
        path: `/api/invites/workshop/email`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description An id needs a signed-in caller: admins see any invite, everyone else the invites addressed to them or created by them. A token also answers an anonymous caller, for open invites.
     *
     * @name GetApiInvitesIdOrToken
     * @summary Returns an invite by its id or its token.
     * @request GET:/api/invites/{idOrToken}
     * @secure
     */
    getApiInvitesIdOrToken: (idOrToken: string, params: RequestParams = {}) =>
      this.request<InviteResp, Error>({
        path: `/api/invites/${idOrToken}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description An anonymous caller accepts a workshop invite by token: this creates a participant account in the browser's language, starts its session, sets the session cookie and answers with the account and its token. A signed-in participant switches to the invite's workshop; a head, staff member or individual enters it and keeps their role; anyone else joins as a participant. An invite by id needs a signed-in caller; an institution invite gives the caller its role.
     *
     * @name PostApiInvitesIdOrTokenAccept
     * @summary Accepts an invite by its id or its token.
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
     * @description Its creator and admins can revoke it, and institution staff a workshop invite.
     *
     * @name DeleteApiInvitesId
     * @summary Revokes a pending invite.
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
     * @description status "declined".
     *
     * @name PostApiInvitesIdDecline
     * @summary Declines an invite and answers with its id and its
     * @request POST:/api/invites/{id}/decline
     * @secure
     */
    postApiInvitesIdDecline: (id: string, params: RequestParams = {}) =>
      this.request<UserRoleInvite, Error>({
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
     * @description Query: search (game name), sortBy (game, model for the AI platform, lastPlayed by default).
     *
     * @name GetApiSessions
     * @summary Lists the caller's recent sessions with their games' names.
     * @request GET:/api/sessions
     * @secure
     */
    getApiSessions: (params: RequestParams = {}) =>
      this.request<UserSessionWithGame[], Error>({
        path: `/api/sessions`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name DeleteApiSessionsId
     * @summary Deletes a session the caller owns and everything it stored.
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
     * @description Only the fields sent change. The default AI quality tier cannot be empty; an empty free-use tier or constraint clears it. Answers with the settings as stored.
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
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description key; admins only. The admin's own API key is used directly. A null apiKeyId clears it.
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
        type: ContentType.Json,
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
     * @description Admins see every user; heads and staff see the members of their institution. Everyone else is refused.
     *
     * @name GetApiUsers
     * @summary Lists the users the caller manages.
     * @request GET:/api/users
     * @secure
     */
    getApiUsers: (params: RequestParams = {}) =>
      this.request<User[], Error>({
        path: `/api/users`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The state is "anonymous" without a sign-in, "registering" for an account whose registration is incomplete, and "registered" with the account and its role. A participant whose workshop is inactive, or an account that is gone, is refused with 403. Every other endpoint refuses an account that is still registering; this one reports it.
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
     * @description the caller's account. Heads, staff and individuals enter a workshop by its id; a null workshopId leaves workshop mode.
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
      this.request<User, Error>({
        path: `/api/users/me/active-workshop`,
        method: "PUT",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description their account. The language is an ISO 639-1 code the platform supports.
     *
     * @name PatchApiUsersMeLanguage
     * @summary Sets the caller's language and answers with
     * @request PATCH:/api/users/me/language
     * @secure
     */
    patchApiUsersMeLanguage: (
      data: UpdateUserLanguageReq,
      params: RequestParams = {},
    ) =>
      this.request<User, Error>({
        path: `/api/users/me/language`,
        method: "PATCH",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Both fields are null for an account without one.
     *
     * @name GetApiUsersMeParticipantCode
     * @summary Returns the caller's own re-login code.
     * @request GET:/api/users/me/participant-code
     * @secure
     */
    getApiUsersMeParticipantCode: (params: RequestParams = {}) =>
      this.request<UserParticipantCodeResp, Error>({
        path: `/api/users/me/participant-code`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name GetApiUsersMeStats
     * @summary Returns the caller's statistics.
     * @request GET:/api/users/me/stats
     * @secure
     */
    getApiUsersMeStats: (params: RequestParams = {}) =>
      this.request<UserStats, Error>({
        path: `/api/users/me/stats`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The name is required. The account's language follows the request's Accept-Language.
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
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description record.
     *
     * @name DeleteApiUsersId
     * @summary Removes a user, such as a participant, keeping their
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
     * @description The caller reads their own account; admins read anyone; heads and staff read the members of their institution.
     *
     * @name GetApiUsersId
     * @summary Returns a user.
     * @request GET:/api/users/{id}
     * @secure
     */
    getApiUsersId: (id: string, params: RequestParams = {}) =>
      this.request<User, Error>({
        path: `/api/users/${id}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The caller updates themselves; admins update anyone; heads and staff update the participants of their institution's workshops. Only admins change an email. An empty name or email leaves it unchanged; an empty aiQualityTier clears it. The age group is "u13" (13-17, no parental consent), "u13p" (13-17, with parental consent) or "u18" (18+).
     *
     * @name PostApiUsersId
     * @summary Updates a user and answers with them.
     * @request POST:/api/users/{id}
     * @secure
     */
    postApiUsersId: (
      id: string,
      data: UpdateUserReq,
      params: RequestParams = {},
    ) =>
      this.request<User, Error>({
        path: `/api/users/${id}`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name DeleteApiUsersIdRole
     * @summary Removes a user's role and answers with the user.
     * @request DELETE:/api/users/{id}/role
     * @secure
     */
    deleteApiUsersIdRole: (id: string, params: RequestParams = {}) =>
      this.request<User, Error>({
        path: `/api/users/${id}/role`,
        method: "DELETE",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The role is scoped to an institution or a workshop by their ids.
     *
     * @name PostApiUsersIdRole
     * @summary Sets a user's role and answers with the user.
     * @request POST:/api/users/{id}/role
     * @secure
     */
    postApiUsersIdRole: (
      id: string,
      data: SetUserRoleReq,
      params: RequestParams = {},
    ) =>
      this.request<User, Error>({
        path: `/api/users/${id}/role`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description Returns the version, git commit and build time the server was built with.
     *
     * @name GetApiVersion
     * @summary Reports the server's build.
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
     * @description one institution. Query: institutionId, search (name), sortBy (name, createdAt, participantCount), sortDir (asc, desc), activeOnly ("true" lists active workshops only).
     *
     * @name GetApiWorkshops
     * @summary Lists the workshops the caller can see, or those of
     * @request GET:/api/workshops
     * @secure
     */
    getApiWorkshops: (params: RequestParams = {}) =>
      this.request<Workshop[], Error>({
        path: `/api/workshops`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name PostApiWorkshops
     * @summary Creates a workshop.
     * @request POST:/api/workshops
     * @secure
     */
    postApiWorkshops: (data: CreateWorkshopReq, params: RequestParams = {}) =>
      this.request<Workshop, Error>({
        path: `/api/workshops`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description staff and heads only.
     *
     * @name GetApiWorkshopsParticipantsParticipantIdToken
     * @summary Returns a workshop participant's access token;
     * @request GET:/api/workshops/participants/{participantId}/token
     * @secure
     */
    getApiWorkshopsParticipantsParticipantIdToken: (
      participantId: string,
      params: RequestParams = {},
    ) =>
      this.request<WorkshopParticipantTokenResp, Error>({
        path: `/api/workshops/participants/${participantId}/token`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description staff and heads only. Earlier sign-in links stop working and the participant's sessions end.
     *
     * @name PostApiWorkshopsParticipantsParticipantIdTokenReset
     * @summary Replaces a workshop participant's access token;
     * @request POST:/api/workshops/participants/{participantId}/token/reset
     * @secure
     */
    postApiWorkshopsParticipantsParticipantIdTokenReset: (
      participantId: string,
      params: RequestParams = {},
    ) =>
      this.request<WorkshopParticipantTokenResp, Error>({
        path: `/api/workshops/participants/${participantId}/token/reset`,
        method: "POST",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * No description
     *
     * @name DeleteApiWorkshopsId
     * @summary Soft-deletes a workshop.
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
     * No description
     *
     * @name GetApiWorkshopsId
     * @summary Returns a workshop.
     * @request GET:/api/workshops/{id}
     * @secure
     */
    getApiWorkshopsId: (id: string, params: RequestParams = {}) =>
      this.request<Workshop, Error>({
        path: `/api/workshops/${id}`,
        method: "GET",
        secure: true,
        format: "json",
        ...params,
      }),

    /**
     * @description The name is required. Omitting publicSlug, publicDescription or publicLinks keeps the current value; an empty description or list clears it.
     *
     * @name PatchApiWorkshopsId
     * @summary Updates a workshop's settings and its public page.
     * @request PATCH:/api/workshops/{id}
     * @secure
     */
    patchApiWorkshopsId: (
      id: string,
      data: UpdateWorkshopReq,
      params: RequestParams = {},
    ) =>
      this.request<Workshop, Error>({
        path: `/api/workshops/${id}`,
        method: "PATCH",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description with; staff and heads only. A null or empty apiKeyShareId removes it.
     *
     * @name PutApiWorkshopsIdApiKey
     * @summary Sets the API key a workshop's participants play
     * @request PUT:/api/workshops/{id}/api-key
     * @secure
     */
    putApiWorkshopsIdApiKey: (
      id: string,
      data: SetWorkshopApiKeyReq,
      params: RequestParams = {},
    ) =>
      this.request<Workshop, Error>({
        path: `/api/workshops/${id}/api-key`,
        method: "PUT",
        body: data,
        secure: true,
        type: ContentType.Json,
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
     * @description making it their active workshop.
     *
     * @name PostApiWorkshopsIdMembers
     * @summary Adds a member of the organization to a workshop by
     * @request POST:/api/workshops/{id}/members
     * @secure
     */
    postApiWorkshopsIdMembers: (
      id: string,
      data: AddWorkshopMemberReq,
      params: RequestParams = {},
    ) =>
      this.request<MessageResp, Error>({
        path: `/api/workshops/${id}/members`,
        method: "POST",
        body: data,
        secure: true,
        type: ContentType.Json,
        format: "json",
        ...params,
      }),

    /**
     * @description workshop by clearing their active workshop.
     *
     * @name DeleteApiWorkshopsIdMembersUserId
     * @summary Removes a member who is not permanent from a
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
  };
}
