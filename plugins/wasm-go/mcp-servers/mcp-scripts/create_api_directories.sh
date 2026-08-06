#!/bin/bash

# Function to display usage
usage() {
    echo "Usage: $0 [options] [api_code1 api_code2 ...]"
    echo "Options:"
    echo "  -h, --help             Display this help message"
    echo ""
    echo "If no api_codes are specified, all APIs will be processed."
    exit 1
}

# Parse command line options
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            ;;
        *)
            # Collect all remaining arguments as API codes
            break
            ;;
    esac
done

# Define the API data with English translations
api_codes=("cmapi00065924" "cmapi00066017" "cmapi00054907" "cmapi012364" "cmapi029030" "cmapi011240" "cmapi011517" "cmapi011178" "cmapi010845" "cmapi011212" "cmapi00045093" "cmapi00067564" "cmapi011529" "cmapi022144" "cmapi026966" "cmapi011221" "cmapi00066410" "cmapi00048162" "cmapi00050817" "cmapi00044839" "cmapi00049059" "cmapi00046930" "cmapi00050226" "cmapi00069588" "cmapi022105" "cmapi00066353" "cmapi00065113" "cmapi027789" "cmapi00062739" "cmapi022081" "cmapi011032" "cmapi011138" "cmapi00066399" "cmapi00047480" "cmapi00067671")
server_names=("stock-helper 股票助手" "calendar-holiday-helper 日历/假期助手" "ip-query ip查询" "weather-query 墨迹天气查询" "business-info-query 工商信息查询" "train-ticket-query 火车票查询" "today-in-history 历史上的今天" "hot-news 热门新闻" "stock-history-data 股票历史数据" "heavenly-stems-and-earthly-branches-query 天干地支查询" "recipe-query 菜谱查询" "business-credit-rating 企业信用评级" "zodiac-analysis 星座分析" "taobao-hot-words 淘宝热词" "fund-data-query 基金数据查询" "exchange-rate-query 汇率查询" "national-bid-query 全国招中标查询" "logistics-tracking-query 物流轨迹查询" "parking-lot-query 停车场查询" "agricultural-product-price-query 农产品价格查询" "business-patent-query 企业专利查询" "vehicle-info-query 车辆信息查询" "invoice-verification 发票查验" "traditional-chinese-medicine-tongue-diagnosis 中医舌诊" "tourist-attraction-query 旅游景点查询" "book-query 图书查询" "route-planning 路径规划" "global-financial-news 全球财经快讯" "oil-price-query 油价查询" "jd-hot-words 京东热词" "product-barcode-query 商品条码查询" "vehicle-restriction-query 车辆限行查询" "resume-analysis 简历解析" "deadbeat-query 老赖查询" "document-conversion 文档转换")

# If specific API codes are provided, filter the arrays
if [[ $# -gt 0 ]]; then
    # Create temporary arrays
    declare -a filtered_api_codes
    declare -a filtered_server_names
    
    for requested_api_code in "$@"; do
        for i in "${!api_codes[@]}"; do
            if [[ "${api_codes[$i]}" == "$requested_api_code" ]]; then
                filtered_api_codes+=("${api_codes[$i]}")
                filtered_server_names+=("${server_names[$i]}")
                break
            fi
        done
    done
    
    # Check if any API codes were found
    if [[ ${#filtered_api_codes[@]} -eq 0 ]]; then
        echo "Error: None of the specified API codes were found"
        exit 1
    fi
    
    # Replace the original arrays with the filtered ones
    api_codes=("${filtered_api_codes[@]}")
    server_names=("${filtered_server_names[@]}")
    
    echo "Processing ${#api_codes[@]} specified API(s)"
else
    echo "Processing all ${#api_codes[@]} APIs"
fi

# Function to process a single API
process_api() {
    local api_code=$1
    local server_name=$2
    local english_name=$(echo "$server_name" | awk '{print $1}')
    local chinese_name=$(echo "$server_name" | awk '{print $2}')
    
    echo "Processing $english_name ($api_code)..."
    
    # Create directory
    mkdir -p "../$english_name"

    # Generate mcp-server.yaml
    $GOPATH/bin/openapi-to-mcp --input "../$english_name/api.json" --output "../$english_name/mcp-server.yaml" --server-name "$english_name" --template yunmarket-tmpl.yaml
    
    # Create README_ZH.md
    echo "# $chinese_name" > "../$english_name/README_ZH.md"
    # Add API details to README.md
    echo -e "\nAPI认证需要的APP Code请在阿里云API市场申请: https://market.aliyun.com/apimarket/detail/$api_code" >> "../$english_name/README_ZH.md"   
    
    # Generate Markdown documentation from YAML and append to README_ZH.md
    if [ -f "../$english_name/mcp-server.yaml" ]; then
        echo -e "\n" >> "../$english_name/README_ZH.md"
        python3 ./yaml_to_markdown.py "../$english_name/mcp-server.yaml" | cat >> "../$english_name/README_ZH.md"
        echo "Generated Markdown documentation for $english_name"
    fi
    
    # Translate README_ZH.md to README.md
    if [ -f "../$english_name/README_ZH.md" ]; then
        python3 ./translate_readme.py "../$english_name/README_ZH.md" "../$english_name/README.md"
        echo "Translated README_ZH.md to README.md for $english_name"
    fi
    
    echo "Completed processing $english_name"
}

# Process APIs sequentially for now (simpler implementation)
for i in "${!api_codes[@]}"; do
    process_api "${api_codes[$i]}" "${server_names[$i]}"
done

echo "All API processing completed"
