import { useQueryClient } from "./QueryClientProvider.js";
import { useIsRestoring } from "./IsRestoringProvider.js";
import { useQueryErrorResetBoundary } from "./QueryErrorResetBoundary.js";
import { ensurePreventErrorBoundaryRetry, getHasError, useClearResetErrorBoundary } from "./errorBoundaryUtils.js";
import { ensureSuspenseTimers, fetchOptimistic, shouldSuspend } from "./suspense.js";
import { useSyncExternalStore } from "./utils.js";
import { QueriesObserver, QueryObserver, noop, notifyManager } from "@tanstack/query-core";
import { useCallback, useEffect, useMemo, useState } from "preact/hooks";
//#region src/useQueries.ts
/**
* The `useQueries` hook can be used to fetch a variable number of queries.
*
* The `queries` key accepts an array with query option objects mostly identical to `useQuery` — see the
* `queries` parameter below for the differences. A custom `QueryClient` is supplied once, as `useQueries`' own
* top-level second argument, rather than per query.
*
* Having the same query key more than once in the array of query objects may cause some data to be shared
* between queries. To avoid this, consider de-duplicating the queries and map the results back to the desired
* structure.
*
* The `combine` option can be used to combine the results of the queries into a single value. The result will
* be structurally shared to be as referentially stable as possible.
*
* @param queryClient - Use this to provide a custom `QueryClient`. Otherwise, the one from the nearest context
* will be used.
* @returns The combined result. Without `combine`, this is an array with all the query results, in the same
* order as the input. When `combine` is provided, this is the value returned by `combine` instead.
*
* @example
* ```tsx
* import { useQueries } from '@tanstack/preact-query'
*
* function Posts({ ids }: { ids: Array<number> }) {
*   const postQueries = useQueries({
*     queries: ids.map((id) => ({
*       queryKey: ['post', id],
*       queryFn: () => fetchPost(id),
*       staleTime: Infinity,
*     })),
*   })
*
*   return (
*     <ul>
*       {postQueries.map((query, index) => {
*         if (query.isPending) return <li key={ids[index]}>Loading...</li>
*         if (query.isError) return <li key={ids[index]}>Error: {query.error.message}</li>
*         return <li key={ids[index]}>{query.data.title}</li>
*       })}
*     </ul>
*   )
* }
* ```
*
* @example
* Combining results into a single value:
* ```tsx
* import { useQueries } from '@tanstack/preact-query'
*
* function Posts({ ids }: { ids: Array<number> }) {
*   const { data, isPending, isError } = useQueries({
*     queries: ids.map((id) => ({
*       queryKey: ['post', id],
*       queryFn: () => fetchPost(id),
*     })),
*     combine: (postQueries) => {
*       return {
*         data: postQueries.map((query) => query.data),
*         isPending: postQueries.some((query) => query.isPending),
*         isError: postQueries.some((query) => query.isError),
*       }
*     },
*   })
*
*   if (isPending) return 'Loading...'
*   if (isError) return 'Error loading posts'
*
*   return (
*     <ul>
*       {data.map((post) => (
*         <li key={post?.id}>{post?.title}</li>
*       ))}
*     </ul>
*   )
* }
* ```
*/
function useQueries({ queries, ...options }, queryClient) {
	const client = useQueryClient(queryClient);
	const isRestoring = useIsRestoring();
	const errorResetBoundary = useQueryErrorResetBoundary();
	const subscribed = options.subscribed !== false;
	const defaultedQueries = useMemo(() => queries.map((opts) => {
		const defaultedOptions = client.defaultQueryOptions(opts);
		defaultedOptions._optimisticResults = isRestoring ? "isRestoring" : subscribed ? "optimistic" : void 0;
		return defaultedOptions;
	}), [
		queries,
		client,
		isRestoring,
		subscribed
	]);
	defaultedQueries.forEach((queryOptions) => {
		ensureSuspenseTimers(queryOptions);
		const query = client.getQueryCache().get(queryOptions.queryHash);
		ensurePreventErrorBoundaryRetry(queryOptions, errorResetBoundary, query);
	});
	useClearResetErrorBoundary(errorResetBoundary);
	const [observer] = useState(() => new QueriesObserver(client, defaultedQueries, options));
	const [optimisticResult, getCombinedResult, trackResult] = observer.getOptimisticResult(defaultedQueries, options.combine);
	const shouldSubscribe = !isRestoring && subscribed;
	useSyncExternalStore(useCallback((onStoreChange) => shouldSubscribe ? observer.subscribe(notifyManager.batchCalls(onStoreChange)) : noop, [observer, shouldSubscribe]), () => observer.getCurrentResult());
	useEffect(() => {
		observer.setQueries(defaultedQueries, options);
	}, [
		defaultedQueries,
		options,
		observer
	]);
	const suspensePromises = optimisticResult.some((result, index) => shouldSuspend(defaultedQueries[index], result)) ? optimisticResult.flatMap((result, index) => {
		const opts = defaultedQueries[index];
		if (opts && shouldSuspend(opts, result)) {
			const queryObserver = new QueryObserver(client, opts);
			return fetchOptimistic(opts, queryObserver, errorResetBoundary);
		}
		return [];
	}) : [];
	if (suspensePromises.length > 0) throw Promise.all(suspensePromises);
	const firstSingleResultWhichShouldThrow = optimisticResult.find((result, index) => {
		const query = defaultedQueries[index];
		return query && getHasError({
			result,
			errorResetBoundary,
			throwOnError: query.throwOnError,
			query: client.getQueryCache().get(query.queryHash),
			suspense: query.suspense
		});
	});
	if (firstSingleResultWhichShouldThrow) throw firstSingleResultWhichShouldThrow.error;
	return getCombinedResult(trackResult());
}
//#endregion
export { useQueries };

//# sourceMappingURL=useQueries.js.map