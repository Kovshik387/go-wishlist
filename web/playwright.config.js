var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
var _a;
import { defineConfig, devices } from "@playwright/test";
export default defineConfig({
    testDir: "./e2e",
    timeout: 30000,
    fullyParallel: false,
    retries: 0,
    reporter: "list",
    use: {
        baseURL: (_a = process.env.E2E_BASE_URL) !== null && _a !== void 0 ? _a : "http://localhost:8080",
        trace: "retain-on-failure",
    },
    projects: [
        {
            name: "mobile-chromium",
            use: __assign({}, devices["iPhone 13"]),
        },
    ],
});
