const U32_MASK64 = BigInt(2 ** 32 - 1);
const _32n = BigInt(32);
export function fromBig(n, le = false) {
    if (le)
        return { h: Number(n & U32_MASK64), l: Number((n >> _32n) & U32_MASK64) };
    return { h: Number((n >> _32n) & U32_MASK64) | 0, l: Number(n & U32_MASK64) | 0 };
}
export function split(lst, le = false) {
    let Ah = new Uint32Array(lst.length);
    let Al = new Uint32Array(lst.length);
    for (let i = 0; i < lst.length; i++) {
        const { h, l } = fromBig(lst[i], le);
        [Ah[i], Al[i]] = [h, l];
    }
    return [Ah, Al];
}
export const toBig = (h, l) => (BigInt(h >>> 0) << _32n) | BigInt(l >>> 0);
// for Shift in [0, 32)
export const shrSH = (h, l, s) => h >>> s;
export const shrSL = (h, l, s) => (h << (32 - s)) | (l >>> s);
// Right rotate for Shift in [1, 32)
export const rotrSH = (h, l, s) => (h >>> s) | (l << (32 - s));
export const rotrSL = (h, l, s) => (h << (32 - s)) | (l >>> s);
// Right rotate for Shift in (32, 64), NOTE: 32 is special case.
export const rotrBH = (h, l, s) => (h << (64 - s)) | (l >>> (s - 32));
export const rotrBL = (h, l, s) => (h >>> (s - 32)) | (l << (64 - s));
// Right rotate for shift===32 (just swaps l&h)
export const rotr32H = (h, l) => l;
export const rotr32L = (h, l) => h;
// Left rotate for Shift in [1, 32)
export const rotlSH = (h, l, s) => (h << s) | (l >>> (32 - s));
export const rotlSL = (h, l, s) => (l << s) | (h >>> (32 - s));
// Left rotate for Shift in (32, 64), NOTE: 32 is special case.
export const rotlBH = (h, l, s) => (l << (s - 32)) | (h >>> (64 - s));
export const rotlBL = (h, l, s) => (h << (s - 32)) | (l >>> (64 - s));
// JS uses 32-bit signed integers for bitwise operations which means we cannot
// simple take carry out of low bit sum by shift, we need to use division.
export function add(Ah, Al, Bh, Bl) {
    const l = (Al >>> 0) + (Bl >>> 0);
    return { h: (Ah + Bh + ((l / 2 ** 32) | 0)) | 0, l: l | 0 };
}
// Addition with more than 2 elements
export const add3L = (Al, Bl, Cl) => (Al >>> 0) + (Bl >>> 0) + (Cl >>> 0);
export const add3H = (low, Ah, Bh, Ch) => (Ah + Bh + Ch + ((low / 2 ** 32) | 0)) | 0;
export const add4L = (Al, Bl, Cl, Dl) => (Al >>> 0) + (Bl >>> 0) + (Cl >>> 0) + (Dl >>> 0);
export const add4H = (low, Ah, Bh, Ch, Dh) => (Ah + Bh + Ch + Dh + ((low / 2 ** 32) | 0)) | 0;
export const add5L = (Al, Bl, Cl, Dl, El) => (Al >>> 0) + (Bl >>> 0) + (Cl >>> 0) + (Dl >>> 0) + (El >>> 0);
export const add5H = (low, Ah, Bh, Ch, Dh, Eh) => (Ah + Bh + Ch + Dh + Eh + ((low / 2 ** 32) | 0)) | 0;
