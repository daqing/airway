import { ComponentChildren } from "preact";
//#region src/QueryErrorResetBoundary.d.ts
/**
 * Resets any query errors within the boundary, so queries know they can try again.
 */
type QueryErrorResetFunction = () => void;
/**
 * Returns whether the boundary has been reset and not yet cleared.
 */
type QueryErrorIsResetFunction = () => boolean;
/**
 * Clears the reset state, so queries know not to try again until the boundary is reset again.
 */
type QueryErrorClearResetFunction = () => void;
interface QueryErrorResetBoundaryValue {
  /**
   * Clears the reset state, so queries know not to try again until the boundary is reset again.
   */
  clearReset: QueryErrorClearResetFunction;
  /**
   * Returns whether the boundary has been reset and not yet cleared.
   */
  isReset: QueryErrorIsResetFunction;
  /**
   * Resets any query errors within the boundary, so queries know they can try again.
   */
  reset: QueryErrorResetFunction;
}
/**
 * This hook will reset any query errors within the closest `QueryErrorResetBoundary`. If there is no boundary
 * defined it will reset them globally.
 *
 * @returns The boundary's {@link QueryErrorResetBoundaryValue}.
 *
 * @example
 * ```tsx
 * import { useErrorBoundary } from 'preact/hooks'
 * import type { ComponentChildren } from 'preact'
 * import { useQueryErrorResetBoundary } from '@tanstack/preact-query'
 *
 * function App({ children }: { children: ComponentChildren }) {
 *   const { reset } = useQueryErrorResetBoundary()
 *   const [error, resetError] = useErrorBoundary(() => reset())
 *
 *   if (error) {
 *     return (
 *       <div>
 *         There was an error!
 *         <button onClick={() => resetError()}>Try again</button>
 *       </div>
 *     )
 *   }
 *
 *   return children
 * }
 * ```
 */
declare const useQueryErrorResetBoundary: () => QueryErrorResetBoundaryValue;
/**
 * A render-prop function usable as `children` on `QueryErrorResetBoundary`.
 *
 * @param value - The boundary's {@link QueryErrorResetBoundaryValue}.
 * @returns The children to render.
 */
type QueryErrorResetBoundaryFunction = (value: QueryErrorResetBoundaryValue) => ComponentChildren;
/**
 * The props accepted by `QueryErrorResetBoundary`.
 */
interface QueryErrorResetBoundaryProps {
  /**
   * Either a plain node, or a function that receives the boundary's {@link QueryErrorResetBoundaryValue} and
   * returns a node.
   */
  children: QueryErrorResetBoundaryFunction | ComponentChildren;
}
/**
 * When using `suspense` or `throwOnError` in your queries, you need a way to let queries know that you want to
 * try again when re-rendering after some error occurred. With the `QueryErrorResetBoundary` component you can
 * reset any query errors within the boundaries of the component.
 *
 * @returns The `children`, rendered as-is, or called with the boundary's {@link QueryErrorResetBoundaryValue}
 * if `children` is a function.
 *
 * @example
 * ```tsx
 * import { useErrorBoundary } from 'preact/hooks'
 * import type { ComponentChildren } from 'preact'
 * import { QueryErrorResetBoundary } from '@tanstack/preact-query'
 *
 * function App() {
 *   return (
 *     <QueryErrorResetBoundary>
 *       {({ reset }) => (
 *         <ErrorBoundary reset={reset}>
 *           <Page />
 *         </ErrorBoundary>
 *       )}
 *     </QueryErrorResetBoundary>
 *   )
 * }
 *
 * function ErrorBoundary({
 *   children,
 *   reset,
 * }: {
 *   children: ComponentChildren
 *   reset: () => void
 * }) {
 *   const [error, resetError] = useErrorBoundary(() => reset())
 *
 *   if (error) {
 *     return (
 *       <div>
 *         There was an error!
 *         <button onClick={() => resetError()}>Try again</button>
 *       </div>
 *     )
 *   }
 *
 *   return children
 * }
 * ```
 */
declare const QueryErrorResetBoundary: ({ children }: QueryErrorResetBoundaryProps) => import("preact").JSX.Element;
//#endregion
export { QueryErrorClearResetFunction, QueryErrorIsResetFunction, QueryErrorResetBoundary, QueryErrorResetBoundaryFunction, QueryErrorResetBoundaryProps, QueryErrorResetBoundaryValue, QueryErrorResetFunction, useQueryErrorResetBoundary };
//# sourceMappingURL=QueryErrorResetBoundary.d.cts.map