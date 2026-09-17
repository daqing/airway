import { createContext } from "preact";
import { useContext, useState } from "preact/hooks";
import { jsx } from "preact/jsx-runtime";
//#region src/QueryErrorResetBoundary.tsx
function createValue() {
	let isReset = false;
	return {
		clearReset: () => {
			isReset = false;
		},
		reset: () => {
			isReset = true;
		},
		isReset: () => {
			return isReset;
		}
	};
}
const QueryErrorResetBoundaryContext = createContext(createValue());
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
const useQueryErrorResetBoundary = () => useContext(QueryErrorResetBoundaryContext);
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
const QueryErrorResetBoundary = ({ children }) => {
	const [value] = useState(() => createValue());
	return /* @__PURE__ */ jsx(QueryErrorResetBoundaryContext.Provider, {
		value,
		children: typeof children === "function" ? children(value) : children
	});
};
//#endregion
export { QueryErrorResetBoundary, useQueryErrorResetBoundary };

//# sourceMappingURL=QueryErrorResetBoundary.js.map