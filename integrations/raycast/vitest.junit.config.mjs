import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    include: ["test/**/*.test.ts"],
    outputFile: {
      junit: "../../reports/junit/raycast.xml",
    },
    reporters: [
      "default",
      [
        "junit",
        {
          addFileAttribute: true,
          classnameTemplate: "{filename}",
          suiteName: "raycast",
        },
      ],
    ],
    testTimeout: 30000,
  },
});
