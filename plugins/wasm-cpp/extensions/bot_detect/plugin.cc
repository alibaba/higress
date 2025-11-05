// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

#include "extensions/bot_detect/plugin.h"

#include <array>
#include <memory>
#include <stdexcept>
#include <string_view>

#include "absl/strings/str_cat.h"
#include "absl/strings/str_join.h"
#include "absl/strings/str_split.h"
#include "common/json_util.h"

using ::nlohmann::json;
using ::Wasm::Common::JsonArrayIterate;
using ::Wasm::Common::JsonGetField;
using ::Wasm::Common::JsonObjectIterate;
using ::Wasm::Common::JsonValueAs;

#ifdef NULL_PLUGIN

namespace proxy_wasm {
namespace null_plugin {
namespace bot_detect {

PROXY_WASM_NULL_PLUGIN_REGISTRY

#endif

static RegisterContextFactory register_BotDetect(
    CONTEXT_FACTORY(PluginContext), ROOT_FACTORY(PluginRootContext));

static std::array<std::string, 6> default_bot_regex = {
    R"(/((?:Ant-)?Nutch|[A-z]+[Bb]ot|[A-z]+[Ss]pider|Axtaris|fetchurl|Isara|ShopSalad|Tailsweep)[ \-](\d+)(?:\.(\d+)(?:\.(\d+))?)?)",
    R"((?:\/[A-Za-z0-9\.]+|) {0,5}([A-Za-z0-9 \-_\!\[\]:]{0,50}(?:[Aa]rchiver|[Ii]ndexer|[Ss]craper|[Bb]ot|[Ss]pider|[Cc]rawl[a-z]{0,50}))[/ ](\d+)(?:\.(\d+)(?:\.(\d+)|)|))",
    R"((?:\/[A-Za-z0-9\.]+|) {0,5}([A-Za-z0-9 \-_\!\[\]:]{0,50}(?:[Aa]rchiver|[Ii]ndexer|[Ss]craper|[Bb]ot|[Ss]pider|[Cc]rawl[a-z]{0,50})) (\d+)(?:\.(\d+)(?:\.(\d+)|)|))",
    R"(((?:[A-z0-9]{1,50}|[A-z\-]{1,50} ?|)(?: the |)(?:[Ss][Pp][Ii][Dd][Ee][Rr]|[Ss]crape|[Cc][Rr][Aa][Ww][Ll])[A-z0-9]{0,50})(?:(?:[ /]| v)(\d+)(?:\.(\d+)|)(?:\.(\d+)|)|))",
    R"(\b(008|Altresium|Argus|BaiduMobaider|BoardReader|DNSGroup|DataparkSearch|EDI|Goodzer|Grub|INGRID|Infohelfer|LinkedInBot|LOOQ|Nutch|OgScrper|PathDefender|Peew|PostPost|Steeler|Twitterbot|VSE|WebCrunch|WebZIP|Y!J-BR[A-Z]|YahooSeeker|envolk|sproose|wminer)/(\d+)(?:\.(\d+)|)(?:\.(\d+)|))",
    R"((CSimpleSpider|Cityreview Robot|CrawlDaddy|CrawlFire|Finderbots|Index crawler|Job Roboter|KiwiStatus Spider|Lijit Crawler|QuerySeekerSpider|ScollSpider|Trends Crawler|USyd-NLP-Spider|SiteCat Webbot|BotName\/\$BotVersion|123metaspider-Bot|1470\.net crawler|50\.nu|8bo Crawler Bot|Aboundex|Accoona-[A-z]{1,30}-Agent|AdsBot-Google(?:-[a-z]{1,30}|)|altavista|AppEngine-Google|archive.{0,30}\.org_bot|archiver|Ask Jeeves|[Bb]ai[Dd]u[Ss]pider(?:-[A-Za-z]{1,30})(?:-[A-Za-z]{1,30}|)|bingbot|BingPreview|blitzbot|BlogBridge|Bloglovin|BoardReader Blog Indexer|BoardReader Favicon Fetcher|boitho.com-dc|BotSeer|BUbiNG|\b\w{0,30}favicon\w{0,30}\b|\bYeti(?:-[a-z]{1,30}|)|Catchpoint(?: bot|)|[Cc]harlotte|Checklinks|clumboot|Comodo HTTP\(S\) Crawler|Comodo-Webinspector-Crawler|ConveraCrawler|CRAWL-E|CrawlConvera|Daumoa(?:-feedfetcher|)|Feed Seeker Bot|Feedbin|findlinks|Flamingo_SearchEngine|FollowSite Bot|furlbot|Genieo|gigabot|GomezAgent|gonzo1|(?:[a-zA-Z]{1,30}-|)Googlebot(?:-[a-zA-Z]{1,30}|)|Google SketchUp|grub-client|gsa-crawler|heritrix|HiddenMarket|holmes|HooWWWer|htdig|ia_archiver|ICC-Crawler|Icarus6j|ichiro(?:/mobile|)|IconSurf|IlTrovatore(?:-Setaccio|)|InfuzApp|Innovazion Crawler|InternetArchive|IP2[a-z]{1,30}Bot|jbot\b|KaloogaBot|Kraken|Kurzor|larbin|LEIA|LesnikBot|Linguee Bot|LinkAider|LinkedInBot|Lite Bot|Llaut|lycos|Mail\.RU_Bot|masscan|masidani_bot|Mediapartners-Google|Microsoft .{0,30} Bot|mogimogi|mozDex|MJ12bot|msnbot(?:-media {0,2}|)|msrbot|Mtps Feed Aggregation System|netresearch|Netvibes|NewsGator[^/]{0,30}|^NING|Nutch[^/]{0,30}|Nymesis|ObjectsSearch|OgScrper|Orbiter|OOZBOT|PagePeeker|PagesInventory|PaxleFramework|Peeplo Screenshot Bot|PlantyNet_WebRobot|Pompos|Qwantify|Read%20Later|Reaper|RedCarpet|Retreiver|Riddler|Rival IQ|scooter|Scrapy|Scrubby|searchsight|seekbot|semanticdiscovery|SemrushBot|Simpy|SimplePie|SEOstats|SimpleRSS|SiteCon|Slackbot-LinkExpanding|Slack-ImgProxy|Slurp|snappy|Speedy Spider|Squrl Java|Stringer|TheUsefulbot|ThumbShotsBot|Thumbshots\.ru|Tiny Tiny RSS|Twitterbot|WhatsApp|URL2PNG|Vagabondo|VoilaBot|^vortex|Votay bot|^voyager|WASALive.Bot|Web-sniffer|WebThumb|WeSEE:[A-z]{1,30}|WhatWeb|WIRE|WordPress|Wotbox|www\.almaden\.ibm\.com|Xenu(?:.s|) Link Sleuth|Xerka [A-z]{1,30}Bot|yacy(?:bot|)|YahooSeeker|Yahoo! Slurp|Yandex\w{1,30}|YodaoBot(?:-[A-z]{1,30}|)|YottaaMonitor|Yowedo|^Zao|^Zao-Crawler|ZeBot_www\.ze\.bz|ZooShot|ZyBorg)(?:[ /]v?(\d+)(?:\.(\d+)(?:\.(\d+)|)|)|))",
};

bool PluginRootContext::parsePluginConfig(const json& configuration,
                                          BotDetectConfigRule& rule) {
  auto it = configuration.find("blocked_code");
  if (it != configuration.end()) {
    auto blocked_code = JsonValueAs<int64_t>(it.value());
    if (blocked_code.second != Wasm::Common::JsonParserResultDetail::OK) {
      LOG_WARN("cannot parse status code");
      return false;
    }
    rule.blocked_code = blocked_code.first.value();
  }
  it = configuration.find("blocked_message");
  if (it != configuration.end()) {
    auto blocked_message = JsonValueAs<std::string>(it.value());
    if (blocked_message.second != Wasm::Common::JsonParserResultDetail::OK) {
      LOG_WARN("cannot parse blocked_message");
      return false;
    }
    rule.blocked_message = blocked_message.first.value();
  }
  if (!JsonArrayIterate(configuration, "allow", [&](const json& item) -> bool {
        auto regex = JsonValueAs<std::string>(item);
        if (regex.second != Wasm::Common::JsonParserResultDetail::OK) {
          LOG_WARN("cannot parse allow");
          return false;
        }
        auto re = std::make_unique<ReMatcher>(regex.first.value());
        if (!re->error().empty()) {
          LOG_WARN(re->error());
          return false;
        }
        rule.allow.push_back(std::move(re));

        return true;
      })) {
    LOG_WARN("failed to parse configuration for allow.");
    return false;
  }
  if (!JsonArrayIterate(configuration, "deny", [&](const json& item) -> bool {
        auto regex = JsonValueAs<std::string>(item);
        if (regex.second != Wasm::Common::JsonParserResultDetail::OK) {
          LOG_WARN("cannot parse deny");
          return false;
        }
        auto re = std::make_unique<ReMatcher>(regex.first.value());
        if (!re->error().empty()) {
          LOG_WARN(re->error());
          return false;
        }
        rule.deny.push_back(std::move(re));

        return true;
      })) {
    LOG_WARN("failed to parse configuration for deny.");
    return false;
  }
  return true;
}

bool PluginRootContext::onConfigure(size_t size) {
  // Parse configuration JSON string.
  if (size > 0 && !configure(size)) {
    LOG_WARN("configuration has errors initialization will not continue.");
    return false;
  }
  if (size == 0) {
    // support empty config
    setEmptyGlobalConfig();
  }
  for (auto& regex : default_bot_regex) {
    default_matchers_.push_back(std::make_unique<ReMatcher>(regex, false));
  }
  return true;
}

bool PluginRootContext::configure(size_t configuration_size) {
  auto configuration_data = getBufferBytes(WasmBufferType::PluginConfiguration,
                                           0, configuration_size);
  // Parse configuration JSON string.
  auto result = ::Wasm::Common::JsonParse(configuration_data->view());
  if (!result.has_value()) {
    LOG_WARN(absl::StrCat("cannot parse plugin configuration JSON string: ",
                          configuration_data->view()));
    return false;
  }
  if (!parseRuleConfig(result.value())) {
    LOG_WARN(absl::StrCat("cannot parse plugin configuration JSON string: ",
                          configuration_data->view()));
    return false;
  }
  return true;
}

bool PluginRootContext::checkHeader(const BotDetectConfigRule& rule) {
  GET_HEADER_VIEW(Wasm::Common::Http::Header::UserAgent, user_agent);
  for (const auto& matcher : rule.allow) {
    if (matcher->match(user_agent)) {
      LOG_DEBUG("bot detected by allow rule");
      return true;
    }
  }
  for (const auto& matcher : rule.deny) {
    if (matcher->match(user_agent)) {
      LOG_DEBUG("bot detected by deny rule");
      sendLocalResponse(rule.blocked_code, "", rule.blocked_message, {});
      return false;
    }
  }
  for (const auto& matcher : default_matchers_) {
    if (matcher->match(user_agent)) {
      LOG_DEBUG("bot detected by default rule");
      sendLocalResponse(rule.blocked_code, "", rule.blocked_message, {});
      return false;
    }
  }
  return true;
}

FilterHeadersStatus PluginContext::onRequestHeaders(uint32_t, bool) {
  auto* rootCtx = rootContext();
  return rootCtx->checkRule([rootCtx](const auto& config) {
    return rootCtx->checkHeader(config);
  })
             ? FilterHeadersStatus::Continue
             : FilterHeadersStatus::StopIteration;
}

#ifdef NULL_PLUGIN

}  // namespace bot_detect
}  // namespace null_plugin
}  // namespace proxy_wasm

#endif
