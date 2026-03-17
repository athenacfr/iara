//# hash=804aed459eaddd5f08a3772d9084f6d1
//# sourceMappingURL=mode-select.test.js.map

function asyncGeneratorStep(gen, resolve, reject, _next, _throw, key, arg) {
    try {
        var info = gen[key](arg);
        var value = info.value;
    } catch (error) {
        reject(error);
        return;
    }
    if (info.done) {
        resolve(value);
    } else {
        Promise.resolve(value).then(_next, _throw);
    }
}
function _async_to_generator(fn) {
    return function() {
        var self = this, args = arguments;
        return new Promise(function(resolve, reject) {
            var gen = fn.apply(self, args);
            function _next(value) {
                asyncGeneratorStep(gen, resolve, reject, _next, _throw, "next", value);
            }
            function _throw(err) {
                asyncGeneratorStep(gen, resolve, reject, _next, _throw, "throw", err);
            }
            _next(undefined);
        });
    };
}
function _ts_generator(thisArg, body) {
    var f, y, t, _ = {
        label: 0,
        sent: function() {
            if (t[0] & 1) throw t[1];
            return t[1];
        },
        trys: [],
        ops: []
    }, g = Object.create((typeof Iterator === "function" ? Iterator : Object).prototype), d = Object.defineProperty;
    return d(g, "next", {
        value: verb(0)
    }), d(g, "throw", {
        value: verb(1)
    }), d(g, "return", {
        value: verb(2)
    }), typeof Symbol === "function" && d(g, Symbol.iterator, {
        value: function() {
            return this;
        }
    }), g;
    function verb(n) {
        return function(v) {
            return step([
                n,
                v
            ]);
        };
    }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while(g && (g = 0, op[0] && (_ = 0)), _)try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [
                op[0] & 2,
                t.value
            ];
            switch(op[0]){
                case 0:
                case 1:
                    t = op;
                    break;
                case 4:
                    _.label++;
                    return {
                        value: op[1],
                        done: false
                    };
                case 5:
                    _.label++;
                    y = op[1];
                    op = [
                        0
                    ];
                    continue;
                case 7:
                    op = _.ops.pop();
                    _.trys.pop();
                    continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) {
                        _ = 0;
                        continue;
                    }
                    if (op[0] === 3 && (!t || op[1] > t[0] && op[1] < t[3])) {
                        _.label = op[1];
                        break;
                    }
                    if (op[0] === 6 && _.label < t[1]) {
                        _.label = t[1];
                        t = op;
                        break;
                    }
                    if (t && _.label < t[2]) {
                        _.label = t[2];
                        _.ops.push(op);
                        break;
                    }
                    if (t[2]) _.ops.pop();
                    _.trys.pop();
                    continue;
            }
            op = body.call(thisArg, _);
        } catch (e) {
            op = [
                6,
                e
            ];
            y = 0;
        } finally{
            f = t = 0;
        }
        if (op[0] & 5) throw op[1];
        return {
            value: op[0] ? op[1] : void 0,
            done: true
        };
    }
}
import { test, expect } from "@microsoft/tui-test";
import { createTestEnv, createFakeProject, IARA_BIN, waitForReady } from "./helpers.js";
var env = createTestEnv();
createFakeProject(env.projectsDir, "mode-test-project", {
    repos: [
        "app"
    ],
    metadata: {
        title: "Mode Test",
        description: "For mode select tests"
    }
});
test.use({
    program: {
        file: IARA_BIN
    },
    rows: 24,
    columns: 80,
    env: env.env
});
// Navigate to mode select screen (project list → task select → mode select)
function goToModeSelect(terminal) {
    return _async_to_generator(function() {
        return _ts_generator(this, function(_state) {
            switch(_state.label){
                case 0:
                    return [
                        4,
                        waitForReady(terminal)
                    ];
                case 1:
                    _state.sent();
                    return [
                        4,
                        expect(terminal.getByText("mode-test-project", {
                            strict: false
                        })).toBeVisible()
                    ];
                case 2:
                    _state.sent();
                    terminal.submit();
                    // Navigate through task select screen - select default branch (item 2)
                    return [
                        4,
                        expect(terminal.getByText("TASKS")).toBeVisible()
                    ];
                case 3:
                    _state.sent();
                    terminal.keyDown();
                    terminal.submit();
                    return [
                        4,
                        expect(terminal.getByText("MODE")).toBeVisible()
                    ];
                case 4:
                    _state.sent();
                    return [
                        2
                    ];
            }
        });
    })();
}
test.describe("Mode Select", function() {
    test("shows MODE header after selecting project", function(param) {
        var terminal = param.terminal;
        return _async_to_generator(function() {
            return _ts_generator(this, function(_state) {
                switch(_state.label){
                    case 0:
                        return [
                            4,
                            goToModeSelect(terminal)
                        ];
                    case 1:
                        _state.sent();
                        return [
                            2
                        ];
                }
            });
        })();
    });
    test("shows SESSIONS section", function(param) {
        var terminal = param.terminal;
        return _async_to_generator(function() {
            return _ts_generator(this, function(_state) {
                switch(_state.label){
                    case 0:
                        return [
                            4,
                            goToModeSelect(terminal)
                        ];
                    case 1:
                        _state.sent();
                        return [
                            4,
                            expect(terminal.getByText("SESSIONS")).toBeVisible()
                        ];
                    case 2:
                        _state.sent();
                        return [
                            2
                        ];
                }
            });
        })();
    });
    test("shows New Session option", function(param) {
        var terminal = param.terminal;
        return _async_to_generator(function() {
            return _ts_generator(this, function(_state) {
                switch(_state.label){
                    case 0:
                        return [
                            4,
                            goToModeSelect(terminal)
                        ];
                    case 1:
                        _state.sent();
                        return [
                            4,
                            expect(terminal.getByText(/New Session/g, {
                                strict: false
                            })).toBeVisible()
                        ];
                    case 2:
                        _state.sent();
                        return [
                            2
                        ];
                }
            });
        })();
    });
    test("shows available modes", function(param) {
        var terminal = param.terminal;
        return _async_to_generator(function() {
            return _ts_generator(this, function(_state) {
                switch(_state.label){
                    case 0:
                        return [
                            4,
                            goToModeSelect(terminal)
                        ];
                    case 1:
                        _state.sent();
                        return [
                            4,
                            expect(terminal.getByText(/code/g, {
                                strict: false
                            })).toBeVisible()
                        ];
                    case 2:
                        _state.sent();
                        return [
                            2
                        ];
                }
            });
        })();
    });
    test("switches mode with right arrow", function(param) {
        var terminal = param.terminal;
        return _async_to_generator(function() {
            return _ts_generator(this, function(_state) {
                switch(_state.label){
                    case 0:
                        return [
                            4,
                            goToModeSelect(terminal)
                        ];
                    case 1:
                        _state.sent();
                        terminal.keyRight();
                        return [
                            4,
                            expect(terminal).toMatchSnapshot()
                        ];
                    case 2:
                        _state.sent();
                        return [
                            2
                        ];
                }
            });
        })();
    });
    test("shows permission toggle hint", function(param) {
        var terminal = param.terminal;
        return _async_to_generator(function() {
            return _ts_generator(this, function(_state) {
                switch(_state.label){
                    case 0:
                        return [
                            4,
                            goToModeSelect(terminal)
                        ];
                    case 1:
                        _state.sent();
                        return [
                            4,
                            expect(terminal.getByText(/permissions/g, {
                                strict: false
                            })).toBeVisible()
                        ];
                    case 2:
                        _state.sent();
                        return [
                            2
                        ];
                }
            });
        })();
    });
    test("toggles permissions with tab", function(param) {
        var terminal = param.terminal;
        return _async_to_generator(function() {
            return _ts_generator(this, function(_state) {
                switch(_state.label){
                    case 0:
                        return [
                            4,
                            goToModeSelect(terminal)
                        ];
                    case 1:
                        _state.sent();
                        // Default is "skip permissions" (bypass=true from global settings)
                        return [
                            4,
                            expect(terminal.getByText(/skip permissions/g)).toBeVisible()
                        ];
                    case 2:
                        _state.sent();
                        terminal.keyPress("Tab");
                        return [
                            4,
                            expect(terminal.getByText(/normal permissions/g)).toBeVisible()
                        ];
                    case 3:
                        _state.sent();
                        return [
                            2
                        ];
                }
            });
        })();
    });
    test("returns to task select on Escape", function(param) {
        var terminal = param.terminal;
        return _async_to_generator(function() {
            return _ts_generator(this, function(_state) {
                switch(_state.label){
                    case 0:
                        return [
                            4,
                            goToModeSelect(terminal)
                        ];
                    case 1:
                        _state.sent();
                        terminal.keyEscape();
                        return [
                            4,
                            expect(terminal.getByText("TASKS")).toBeVisible()
                        ];
                    case 2:
                        _state.sent();
                        return [
                            2
                        ];
                }
            });
        })();
    });
    test("launches on Enter (exits TUI)", function(param) {
        var terminal = param.terminal;
        return _async_to_generator(function() {
            return _ts_generator(this, function(_state) {
                switch(_state.label){
                    case 0:
                        return [
                            4,
                            goToModeSelect(terminal)
                        ];
                    case 1:
                        _state.sent();
                        return [
                            4,
                            expect(terminal.getByText(/New Session/g, {
                                strict: false
                            })).toBeVisible()
                        ];
                    case 2:
                        _state.sent();
                        terminal.submit();
                        terminal.onExit(function(exit) {
                            expect(exit.exitCode).toBe(0);
                        });
                        return [
                            2
                        ];
                }
            });
        })();
    });
});
