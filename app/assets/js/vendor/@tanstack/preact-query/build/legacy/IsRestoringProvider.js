import { createContext } from "preact";
import { useContext } from "preact/hooks";
//#region src/IsRestoringProvider.ts
const IsRestoringContext = createContext(false);
/**
* If you are using `PersistQueryClientProvider`, you can also use the `useIsRestoring` hook alongside it to
* check if a restore is currently in progress. `useQuery` and friends also check this internally to avoid
* race conditions between the restore and mounting queries.
*
* @returns `true` while a persisted client is being restored, `false` otherwise.
*/
const useIsRestoring = () => useContext(IsRestoringContext);
/**
* The Provider that `PersistQueryClientProvider` uses to signal whether a persisted client is currently
* being restored, read by `useIsRestoring`.
*/
const IsRestoringProvider = IsRestoringContext.Provider;
//#endregion
export { IsRestoringProvider, useIsRestoring };

//# sourceMappingURL=IsRestoringProvider.js.map