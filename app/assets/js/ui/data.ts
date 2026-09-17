import { useQuery } from "@tanstack/preact-query";

// The lib/render JSON envelope: {"code":0,"data":…,"message":""}. code 0 is
// success; anything else is a business error carried over HTTP 200.

export type ApiResponse<T> = { code: number; data: T; message: string };

export class ApiError extends Error {
  code: number;
  constructor(code: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.code = code;
  }
}

/** Global hook for business errors (e.g. route to a login page on 401). */
let onApiError: ((err: ApiError) => void) | null = null;

export function setApiErrorHandler(fn: ((err: ApiError) => void) | null) {
  onApiError = fn;
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(path, init);
  } catch (e) {
    throw new ApiError(-1, `network error: ${(e as Error).message}`);
  }

  let body: ApiResponse<T> | null = null;
  try {
    body = (await res.json()) as ApiResponse<T>;
  } catch {
    // non-JSON response (proxy error page, 502 html, …)
  }

  if (!res.ok) {
    throw new ApiError(res.status, body?.message || `HTTP ${res.status}`);
  }
  if (!body) {
    throw new ApiError(-1, "empty response body");
  }
  if (body.code !== 0) {
    const err = new ApiError(body.code, body.message || "request failed");
    onApiError?.(err);
    throw err;
  }
  return body.data;
}

/** useQuery over apiFetch — see ui/data usage in the showcase island. */
export function useApiQuery<T>(key: unknown[], path: string) {
  return useQuery<T>({
    queryKey: key,
    queryFn: () => apiFetch<T>(path),
  });
}
