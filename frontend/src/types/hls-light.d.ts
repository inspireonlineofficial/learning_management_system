// hls.js ships types for the full build only; the "light" build (no
// subtitles, alternate audio or EME/DRM — none of which the LMS uses) has the
// same API and is ~35% smaller.
declare module "hls.js/light" {
  export * from "hls.js";
  export { default } from "hls.js";
}
