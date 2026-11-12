import type { Hands, HandsConfig } from "@mediapipe/hands";
import handsScript from "@mediapipe/hands/hands.js?url";
import graph from "@mediapipe/hands/hands.binarypb?url&no-inline";
import fullModel from "@mediapipe/hands/hand_landmark_full.tflite?url";
import liteModel from "@mediapipe/hands/hand_landmark_lite.tflite?url";
import assetLoader from "@mediapipe/hands/hands_solution_packed_assets_loader.js?url";
import packedAssets from "@mediapipe/hands/hands_solution_packed_assets.data?url";
import simdScript from "@mediapipe/hands/hands_solution_simd_wasm_bin.js?url";
import simdWasm from "@mediapipe/hands/hands_solution_simd_wasm_bin.wasm?url";
import wasmScript from "@mediapipe/hands/hands_solution_wasm_bin.js?url";
import wasm from "@mediapipe/hands/hands_solution_wasm_bin.wasm?url";

type HandsConstructor = new (config?: HandsConfig) => Hands;
const browser = globalThis as typeof globalThis & { Hands?: HandsConstructor };

const assets: Record<string, string> = {
  "hands.binarypb": graph,
  "hand_landmark_full.tflite": fullModel,
  "hand_landmark_lite.tflite": liteModel,
  "hands_solution_packed_assets_loader.js": assetLoader,
  "hands_solution_packed_assets.data": packedAssets,
  "hands_solution_simd_wasm_bin.js": simdScript,
  "hands_solution_simd_wasm_bin.wasm": simdWasm,
  "hands_solution_wasm_bin.js": wasmScript,
  "hands_solution_wasm_bin.wasm": wasm,
};

let loadingScript: Promise<HandsConstructor> | undefined;

export async function createHands(): Promise<Hands> {
  // MediaPipe Hands is a classic script. Its package marks imports as side-effect-free.
  loadingScript ??= new Promise<HandsConstructor>((resolve, reject) => {
    const script = document.createElement("script");
    script.src = handsScript;
    script.onload = () => {
      if (browser.Hands) resolve(browser.Hands);
      else reject(new Error("MediaPipe Hands did not initialize"));
    };
    script.onerror = () => reject(new Error("Could not load MediaPipe Hands"));
    document.head.append(script);
  }).catch((error) => {
    loadingScript = undefined;
    throw error;
  });

  const Hands = await loadingScript;
  return new Hands({
    locateFile(file) {
      const url = assets[file];
      if (!url) throw new Error(`Unknown MediaPipe asset: ${file}`);
      return url;
    },
  });
}
