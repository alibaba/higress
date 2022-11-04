#include <assert.h>

#include <string>
#include <unordered_map>

#include "common/http_util.h"
#include "common/route_rule_matcher.h"
#define ASSERT(_X) assert(_X)

#ifndef NULL_PLUGIN

#include "proxy_wasm_intrinsics.h"

#else

#include "include/proxy-wasm/null_plugin.h"

namespace proxy_wasm {
namespace null_plugin {
namespace custom_response {

#endif

struct CustomResponseConfigRule {
  std::vector<std::string> enable_on_status;
  std::vector<std::pair<std::string, std::string>> headers;
  std::string content_type;
  int32_t status_code = 200;
  std::string body;
};

// PluginRootContext is the root context for all streams processed by the
// thread. It has the same lifetime as the worker thread and acts as target for
// interactions that outlives individual stream, e.g. timer, async calls.
class PluginRootContext : public RootContext,
                          public RouteRuleMatcher<CustomResponseConfigRule> {
 public:
  PluginRootContext(uint32_t id, std::string_view root_id)
      : RootContext(id, root_id) {}
  ~PluginRootContext() {}
  bool onConfigure(size_t) override;
  FilterHeadersStatus onRequest(const CustomResponseConfigRule&);
  FilterHeadersStatus onResponse(const CustomResponseConfigRule&);
  bool configure(size_t);

 private:
  bool parsePluginConfig(const json&, CustomResponseConfigRule&) override;
};

// Per-stream context.
class PluginContext : public Context {
 public:
  explicit PluginContext(uint32_t id, RootContext* root) : Context(id, root) {}
  FilterHeadersStatus onRequestHeaders(uint32_t, bool) override;
  FilterHeadersStatus onResponseHeaders(uint32_t, bool) override;

 private:
  inline PluginRootContext* rootContext() {
    return dynamic_cast<PluginRootContext*>(this->root());
  }
};

#ifdef NULL_PLUGIN

}  // namespace custom_response
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
