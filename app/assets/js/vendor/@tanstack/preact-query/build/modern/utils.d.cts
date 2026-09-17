//#region src/utils.d.ts
declare function useSyncExternalStore(subscribe: (onStoreChange: () => void) => () => void, getSnapshot: () => any): any;
declare function useSyncExternalStoreWithSelector<TSnapshot, TSelected>(subscribe: (onStoreChange: () => void) => () => void, getSnapshot: () => TSnapshot, selector: (snapshot: TSnapshot) => TSelected, isEqual: (a: TSelected, b: TSelected) => boolean): TSelected;
//#endregion
export { useSyncExternalStore, useSyncExternalStoreWithSelector };
//# sourceMappingURL=utils.d.cts.map