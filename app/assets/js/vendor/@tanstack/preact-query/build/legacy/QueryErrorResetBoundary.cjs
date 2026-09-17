Object.defineProperty(exports, Symbol.toStringTag, { value: "Module" });
let preact = require("preact");
let preact_hooks = require("preact/hooks");
let preact_jsx_runtime = require("preact/jsx-runtime");
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
const QueryErrorResetBoundaryContext = (0, preact.createContext)(createValue());
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
const useQueryErrorResetBoundary = () => (0, preact_hooks.useContext)(QueryErrorResetBoundaryContext);
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
	const [value] = (0, preact_hooks.useState)(() => createValue());
	return /* @__PURE__ */ (0, preact_jsx_runtime.jsx)(QueryErrorResetBoundaryContext.Provider, {
		value,
		children: typeof children === "function" ? children(value) : children
	});
};
//#endregion
exports.QueryErrorResetBoundary = QueryErrorResetBoundary;
exports.useQueryErrorResetBoundary = useQueryErrorResetBoundary;

//# sourceMappingURL=QueryErrorResetBoundary.cjs.map