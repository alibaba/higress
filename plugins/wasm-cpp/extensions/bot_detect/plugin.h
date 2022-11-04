#include <assert.h>

#include <string>
#include <unordered_map>

#include "common/http_util.h"
#include "common/regex.h"
#include "common/route_rule_matcher.h"
#define ASSERT(_X) assert(_X)

#ifndef NULL_PLUGIN

#include "proxy_wasm_intrinsics.h"

#else

#include "include/proxy-wasm/null_plugin.h"

namespace proxy_wasm {
namespace null_plugin {
namespace bot_detect {

#endif

using ReMatcher = Wasm::Common::Regex::CompiledGoogleReMatcher;
using ReMatcherPtr = std::unique_ptr<ReMatcher>;

struct BotDetectConfigRule {
  int blocked_code = 403;
  std::string blocked_message;
  std::vector<ReMatcherPtr> allow;
  std::vector<ReMatcherPtr> deny;
};

// PluginRootContext is the root context for all streams processed by the
// thread. It has the same lifetime as the worker thread and acts as target for
// interactions that outlives individual stream, e.g. timer, async calls.
class PluginRootContext : public RootContext,
                          public RouteRuleMatcher<BotDetectConfigRule> {
 public:
  PluginRootContext(uint32_t id, std::string_view root_id)
      : RootContext(id, root_id) {}
  ~PluginRootContext() {}
  bool onConfigure(size_t) override;
  bool checkHeader(const BotDetectConfigRule&);
  bool configure(size_t);

 private:
  bool parsePluginConfig(const json&, BotDetectConfigRule&) override;

  std::vector<ReMatcherPtr> default_matchers_;
};

// Per-stream context.
class PluginContext : public Context {
 public:
  explicit PluginContext(uint32_t id, RootContext* root) : Context(id, root) {}
  FilterHeadersStatus onRequestHeaders(uint32_t, bool) override;

 private:
  inline PluginRootContext* rootContext() {
    return dynamic_cast<PluginRootContext*>(this->root());
  }
};

#ifdef NULL_PLUGIN

}  // namespace bot_detect
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
