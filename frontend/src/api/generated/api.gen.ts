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

export type GetCurrentUserReq = object;

export interface GetCurrentUserResp {
  state: string;
  user: object;
}

export type GetVersionReq = object;

export interface GetVersionResp {
  buildTime: string;
  gitCommit: string;
  version: string;
}

export interface SchemafAuth0LoginRequest {
  id_token: string;
}

export interface SchemafLoginResponse {
  subject: string;
  token: string;
}

export interface SchemafRefreshResponse {
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
      this.request<SchemafLoginResponse, void>({
        path: `/api/auth/auth0/login`,
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
      this.request<void, void>({
        path: `/api/auth/logout`,
        method: "POST",
        ...params,
      }),

    /**
     * @description Takes {"token": "..."}: the participant token whole, or its words from a link, QR code or typed code. Starts a session, sets the session cookie and answers {"token": "...", "userId": "..."}; the token is for Bearer clients.
     *
     * @name PostApiAuthParticipantLogin
     * @summary Signs a workshop participant in with their token.
     * @request POST:/api/auth/participant-login
     */
    postApiAuthParticipantLogin: (params: RequestParams = {}) =>
      this.request<void, void>({
        path: `/api/auth/participant-login`,
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
      this.request<SchemafRefreshResponse, void>({
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
    postApiAuthRegister: (params: RequestParams = {}) =>
      this.request<void, void>({
        path: `/api/auth/register`,
        method: "POST",
        secure: true,
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
      this.request<SchemafLoginResponse, void>({
        path: `/api/auth/${provider}/login`,
        method: "POST",
        body: data,
        type: ContentType.Json,
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
      this.request<GetCurrentUserResp, void>({
        path: `/api/users/me`,
        method: "GET",
        secure: true,
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
      this.request<GetVersionResp, void>({
        path: `/api/version`,
        method: "GET",
        format: "json",
        ...params,
      }),
  };
}
