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

package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/antchfx/xmlquery"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

const (
	DEFAULT_MAX_BODY_BYTES uint32 = 100 * 1024 * 1024
	fullSearchPrompts      string = `
# 目标
你需要分析**用户发送的消息**，是否需要查询搜索引擎(Google/Bing)/论文资料库(Arxiv)/私有知识库，并按照如下情况回复相应内容:

## 情况一：不需要查询搜索引擎/论文资料/私有知识库
### 情况举例：
1. **用户发送的消息**不是在提问或寻求帮助
2. **用户发送的消息**是要求翻译文字

### 执行指示
根据上面的**情况举例**，如果符合，则按照下面**回复内容**进行回复

### 回复内容：
none

## 情况二：需要查询搜索引擎/论文资料/私有知识库
### 情况举例：
1. 答复**用户发送的消息**，需依赖互联网上最新的资料
2. 答复**用户发送的消息**，需依赖论文等专业资料
3. 通过查询资料，可以更好地答复**用户发送的消息**

### 执行指示
根据上面的**情况举例**，以及其他需要查询资料的情况，如果符合，按照以下步骤进行处理:
1. What: 分析要答复**用户发送的消息**，需要了解什么知识和资料
2. Where: 判断了解这个知识和资料要向Google等搜索引擎提问，还是向Arxiv论文资料库进行查询，还是向私有知识库进行查询，或者需要同时查询多个地方
3. How: 分析对于要查询的知识和资料，应该提出什么样的问题
4. Adjust: 明确要向什么地方查询什么问题后，按下面方式对问题进行调整
  4.1. 向搜索引擎提问：用一句话概括问题，并且针对搜索引擎做问题优化
  4.2. 向私有知识库提问：将问题拆分成多组关键词的组合，同时组合中的关键词个数尽量不要超过3个
  4.3. 向Arxiv论文资料库提问：
    4.3.1. 明确问题所属领域，然后确定Arxiv的Category值，Category可选的枚举如下:
      - cs.AI: Artificial Intelligence
      - cs.AR: Hardware Architecture
      - cs.CC: Computational Complexity
      - cs.CE: Computational Engineering, Finance, and Science
      - cs.CG: Computational Geometry
      - cs.CL: Computation and Language
      - cs.CR: Cryptography and Security
      - cs.CV: Computer Vision and Pattern Recognition
      - cs.CY: Computers and Society
      - cs.DB: Databases则按照下面**回复内容**进行回复
      - cs.DC: Distributed, Parallel, and Cluster Computing
      - cs.DL: Digital Libraries
      - cs.DM: Discrete Mathematics
      - cs.DS: Data Structures and Algorithms
      - cs.ET: Emerging Technologies
      - cs.FL: Formal Languages and Automata Theory
      - cs.GL: General Literature
      - cs.GR: Graphics
      - cs.GT: Computer Science and Game Theory
      - cs.HC: Human-Computer Interaction
      - cs.IR: Information Retrieval
      - cs.IT: Information Theory
      - cs.LG: Machine Learning
      - cs.LO: Logic in Computer Science
      - cs.MA: Multiagent Systems
      - cs.MM: Multimedia
      - cs.MS: Mathematical Software
      - cs.NA: Numerical Analysis
      - cs.NE: Neural and Evolutionary Computing
      - cs.NI: Networking and Internet Architecture
      - cs.OH: Other Computer Science
      - cs.OS: Operating Systems
      - cs.PF: Performance
      - cs.PL: Programming Languages
      - cs.RO: Robotics
      - cs.SC: Symbolic Computation
      - cs.SD: Sound
      - cs.SE: Software Engineering
      - cs.SI: Social and Information Networks
      - cs.SY: Systems and Control
      - econ.EM: Econometrics
      - econ.GN: General Economics
      - econ.TH: Theoretical Economics
      - eess.AS: Audio and Speech Processing
      - eess.IV: Image and Video Processing
      - eess.SP: Signal Processing
      - eess.SY: Systems and Control
      - math.AC: Commutative Algebra
      - math.AG: Algebraic Geometry
      - math.AP: Analysis of PDEs
      - math.AT: Algebraic Topology
      - math.CA: Classical Analysis and ODEs
      - math.CO: Combinatorics
      - math.CT: Category Theory
      - math.CV: Complex Variables
      - math.DG: Differential Geometry
      - math.DS: Dynamical Systems
      - math.FA: Functional Analysis
      - math.GM: General Mathematics
      - math.GN: General Topology
      - math.GR: Group Theory
      - math.GT: Geometric Topology
      - math.HO: History and Overview
      - math.IT: Information Theory
      - math.KT: K-Theory and Homology
      - math.LO: Logic
      - math.MG: Metric Geometry
      - math.MP: Mathematical Physics
      - math.NA: Numerical Analysis
      - math.NT: Number Theory
      - math.OA: Operator Algebras
      - math.OC: Optimization and Control
      - math.PR: Probability
      - math.QA: Quantum Algebra
      - math.RA: Rings and Algebras
      - math.RT: Representation Theory
      - math.SG: Symplectic Geometry
      - math.SP: Spectral Theory
      - math.ST: Statistics Theory
      - astro-ph.CO: Cosmology and Nongalactic Astrophysics
      - astro-ph.EP: Earth and Planetary Astrophysics
      - astro-ph.GA: Astrophysics of Galaxies
      - astro-ph.HE: High Energy Astrophysical Phenomena
      - astro-ph.IM: Instrumentation and Methods for Astrophysics
      - astro-ph.SR: Solar and Stellar Astrophysics
      - cond-mat.dis-nn: Disordered Systems and Neural Networks
      - cond-mat.mes-hall: Mesoscale and Nanoscale Physics
      - cond-mat.mtrl-sci: Materials Science
      - cond-mat.other: Other Condensed Matter
      - cond-mat.quant-gas: Quantum Gases
      - cond-mat.soft: Soft Condensed Matter
      - cond-mat.stat-mech: Statistical Mechanics
      - cond-mat.str-el: Strongly Correlated Electrons
      - cond-mat.supr-con: Superconductivity
      - gr-qc: General Relativity and Quantum Cosmology
      - hep-ex: High Energy Physics - Experiment
      - hep-lat: High Energy Physics - Lattice
      - hep-ph: High Energy Physics - Phenomenology
      - hep-th: High Energy Physics - Theory
      - math-ph: Mathematical Physics
      - nlin.AO: Adaptation and Self-Organizing Systems
      - nlin.CD: Chaotic Dynamics
      - nlin.CG: Cellular Automata and Lattice Gases
      - nlin.PS: Pattern Formation and Solitons
      - nlin.SI: Exactly Solvable and Integrable Systems
      - nucl-ex: Nuclear Experiment
      - nucl-th: Nuclear Theory
      - physics.acc-ph: Accelerator Physics
      - physics.ao-ph: Atmospheric and Oceanic Physics
      - physics.app-ph: Applied Physics
      - physics.atm-clus: Atomic and Molecular Clusters
      - physics.atom-ph: Atomic Physics
      - physics.bio-ph: Biological Physics
      - physics.chem-ph: Chemical Physics
      - physics.class-ph: Classical Physics
      - physics.comp-ph: Computational Physics
      - physics.data-an: Data Analysis, Statistics and Probability
      - physics.ed-ph: Physics Education
      - physics.flu-dyn: Fluid Dynamics
      - physics.gen-ph: General Physics
      - physics.geo-ph: Geophysics
      - physics.hist-ph: History and Philosophy of Physics
      - physics.ins-det: Instrumentation and Detectors
      - physics.med-ph: Medical Physics
      - physics.optics: Optics
      - physics.plasm-ph: Plasma Physics
      - physics.pop-ph: Popular Physics
      - physics.soc-ph: Physics and Society
      - physics.space-ph: Space Physics
      - quant-ph: Quantum Physics
      - q-bio.BM: Biomolecules
      - q-bio.CB: Cell Behavior
      - q-bio.GN: Genomics
      - q-bio.MN: Molecular Networks
      - q-bio.NC: Neurons and Cognition
      - q-bio.OT: Other Quantitative Biology
      - q-bio.PE: Populations and Evolution
      - q-bio.QM: Quantitative Methods
      - q-bio.SC: Subcellular Processes
      - q-bio.TO: Tissues and Organs
      - q-fin.CP: Computational Finance
      - q-fin.EC: Economics
      - q-fin.GN: General Finance
      - q-fin.MF: Mathematical Finance
      - q-fin.PM: Portfolio Management
      - q-fin.PR: Pricing of Securities
      - q-fin.RM: Risk Management
      - q-fin.ST: Statistical Finance
      - q-fin.TR: Trading and Market Microstructure
      - stat.AP: Applications
      - stat.CO: Computation
      - stat.ME: Methodology
      - stat.ML: Machine Learning
      - stat.OT: Other Statistics
      - stat.TH: Statistics Theory
    4.3.2. 根据问题所属领域，将问题拆分成多组关键词的组合，同时组合中的关键词个数尽量不要超过3个
5. Action: 按照下面**回复内容示例**进行回复，注意:
     - 可以向多个查询目标分别查询多次，多个查询用换行分隔，总查询次数控制在5次以内
     - 查询搜索引擎时，需要以"internet:"开头
     - 查询私有知识库时，需要以"private:"开头
     - 查询Arxiv论文时，需要以Arxiv的Category值开头，例如"cs.AI:"
     - 查询Arxiv论文时，优先用英文表述关键词进行搜索
     - 当用多个关键词查询时，关键词之间用","分隔
     - 尽量满足**用户发送的消息**中的搜索要求，例如用户要求用英文搜索，则需用英文表述问题和关键词
     - 用户如果没有要求搜索语言，则用和**用户发送的消息**一致的语言表述问题和关键词

### 回复内容示例：

#### 用不同语言查询多次搜索引擎
internet: 黄金价格走势
internet: The trend of gold prices

#### 向Arxiv的多个类目查询多次
cs.AI: attention mechanism
cs.AI: neuron
q-bio.NC: brain,attention mechanism

#### 向私有知识库查询多次
private: 电子钱包,密码
private: 张三,身份证号

#### 向多个查询目标查询多次
internet: 中国未来房价趋势
internet: 最新中国经济政策
econ.TH: policy, real estate
private: 财务状况

# 用户发送的消息为：
{question}
`
	arxivSearchPrompts string = `
# 目标
你需要分析**用户发送的消息**，是否需要查询搜索引擎(Google/Bing)/论文资料库(Arxiv)，并按照如下情况回复相应内容:

## 情况一：不需要查询搜索引擎/论文资料
### 情况举例：
1. **用户发送的消息**不是在提问或寻求帮助
2. **用户发送的消息**是要求翻译文字

### 执行指示
根据上面的**情况举例**，如果符合，则按照下面**回复内容**进行回复

### 回复内容：
none

## 情况二：需要查询搜索引擎/论文资料
### 情况举例：
1. 答复**用户发送的消息**，需依赖互联网上最新的资料
2. 答复**用户发送的消息**，需依赖论文等专业资料
3. 通过查询资料，可以更好地答复**用户发送的消息**

### 执行指示
根据上面的**情况举例**，以及其他需要查询资料的情况，如果符合，按照以下步骤进行处理:
1. What: 分析要答复**用户发送的消息**，需要了解什么知识和资料
2. Where: 判断了解这个知识和资料要向Google等搜索引擎提问，还是向Arxiv论文资料库进行查询，或者需要同时查询多个地方
3. How: 分析对于要查询的知识和资料，应该提出什么样的问题
4. Adjust: 明确要向什么地方查询什么问题后，按下面方式对问题进行调整
  4.1. 向搜索引擎提问：用一句话概括问题，并且针对搜索引擎做问题优化
  4.2. 向Arxiv论文资料库提问：
    4.2.1. 明确问题所属领域，然后确定Arxiv的Category值，Category可选的枚举如下:
      - cs.AI: Artificial Intelligence
      - cs.AR: Hardware Architecture
      - cs.CC: Computational Complexity
      - cs.CE: Computational Engineering, Finance, and Science
      - cs.CG: Computational Geometry
      - cs.CL: Computation and Language
      - cs.CR: Cryptography and Security
      - cs.CV: Computer Vision and Pattern Recognition
      - cs.CY: Computers and Society
      - cs.DB: Databases则按照下面**回复内容**进行回复
      - cs.DC: Distributed, Parallel, and Cluster Computing
      - cs.DL: Digital Libraries
      - cs.DM: Discrete Mathematics
      - cs.DS: Data Structures and Algorithms
      - cs.ET: Emerging Technologies
      - cs.FL: Formal Languages and Automata Theory
      - cs.GL: General Literature
      - cs.GR: Graphics
      - cs.GT: Computer Science and Game Theory
      - cs.HC: Human-Computer Interaction
      - cs.IR: Information Retrieval
      - cs.IT: Information Theory
      - cs.LG: Machine Learning
      - cs.LO: Logic in Computer Science
      - cs.MA: Multiagent Systems
      - cs.MM: Multimedia
      - cs.MS: Mathematical Software
      - cs.NA: Numerical Analysis
      - cs.NE: Neural and Evolutionary Computing
      - cs.NI: Networking and Internet Architecture
      - cs.OH: Other Computer Science
      - cs.OS: Operating Systems
      - cs.PF: Performance
      - cs.PL: Programming Languages
      - cs.RO: Robotics
      - cs.SC: Symbolic Computation
      - cs.SD: Sound
      - cs.SE: Software Engineering
      - cs.SI: Social and Information Networks
      - cs.SY: Systems and Control
      - econ.EM: Econometrics
      - econ.GN: General Economics
      - econ.TH: Theoretical Economics
      - eess.AS: Audio and Speech Processing
      - eess.IV: Image and Video Processing
      - eess.SP: Signal Processing
      - eess.SY: Systems and Control
      - math.AC: Commutative Algebra
      - math.AG: Algebraic Geometry
      - math.AP: Analysis of PDEs
      - math.AT: Algebraic Topology
      - math.CA: Classical Analysis and ODEs
      - math.CO: Combinatorics
      - math.CT: Category Theory
      - math.CV: Complex Variables
      - math.DG: Differential Geometry
      - math.DS: Dynamical Systems
      - math.FA: Functional Analysis
      - math.GM: General Mathematics
      - math.GN: General Topology
      - math.GR: Group Theory
      - math.GT: Geometric Topology
      - math.HO: History and Overview
      - math.IT: Information Theory
      - math.KT: K-Theory and Homology
      - math.LO: Logic
      - math.MG: Metric Geometry
      - math.MP: Mathematical Physics
      - math.NA: Numerical Analysis
      - math.NT: Number Theory
      - math.OA: Operator Algebras
      - math.OC: Optimization and Control
      - math.PR: Probability
      - math.QA: Quantum Algebra
      - math.RA: Rings and Algebras
      - math.RT: Representation Theory
      - math.SG: Symplectic Geometry
      - math.SP: Spectral Theory
      - math.ST: Statistics Theory
      - astro-ph.CO: Cosmology and Nongalactic Astrophysics
      - astro-ph.EP: Earth and Planetary Astrophysics
      - astro-ph.GA: Astrophysics of Galaxies
      - astro-ph.HE: High Energy Astrophysical Phenomena
      - astro-ph.IM: Instrumentation and Methods for Astrophysics
      - astro-ph.SR: Solar and Stellar Astrophysics
      - cond-mat.dis-nn: Disordered Systems and Neural Networks
      - cond-mat.mes-hall: Mesoscale and Nanoscale Physics
      - cond-mat.mtrl-sci: Materials Science
      - cond-mat.other: Other Condensed Matter
      - cond-mat.quant-gas: Quantum Gases
      - cond-mat.soft: Soft Condensed Matter
      - cond-mat.stat-mech: Statistical Mechanics
      - cond-mat.str-el: Strongly Correlated Electrons
      - cond-mat.supr-con: Superconductivity
      - gr-qc: General Relativity and Quantum Cosmology
      - hep-ex: High Energy Physics - Experiment
      - hep-lat: High Energy Physics - Lattice
      - hep-ph: High Energy Physics - Phenomenology
      - hep-th: High Energy Physics - Theory
      - math-ph: Mathematical Physics
      - nlin.AO: Adaptation and Self-Organizing Systems
      - nlin.CD: Chaotic Dynamics
      - nlin.CG: Cellular Automata and Lattice Gases
      - nlin.PS: Pattern Formation and Solitons
      - nlin.SI: Exactly Solvable and Integrable Systems
      - nucl-ex: Nuclear Experiment
      - nucl-th: Nuclear Theory
      - physics.acc-ph: Accelerator Physics
      - physics.ao-ph: Atmospheric and Oceanic Physics
      - physics.app-ph: Applied Physics
      - physics.atm-clus: Atomic and Molecular Clusters
      - physics.atom-ph: Atomic Physics
      - physics.bio-ph: Biological Physics
      - physics.chem-ph: Chemical Physics
      - physics.class-ph: Classical Physics
      - physics.comp-ph: Computational Physics
      - physics.data-an: Data Analysis, Statistics and Probability
      - physics.ed-ph: Physics Education
      - physics.flu-dyn: Fluid Dynamics
      - physics.gen-ph: General Physics
      - physics.geo-ph: Geophysics
      - physics.hist-ph: History and Philosophy of Physics
      - physics.ins-det: Instrumentation and Detectors
      - physics.med-ph: Medical Physics
      - physics.optics: Optics
      - physics.plasm-ph: Plasma Physics
      - physics.pop-ph: Popular Physics
      - physics.soc-ph: Physics and Society
      - physics.space-ph: Space Physics
      - quant-ph: Quantum Physics
      - q-bio.BM: Biomolecules
      - q-bio.CB: Cell Behavior
      - q-bio.GN: Genomics
      - q-bio.MN: Molecular Networks
      - q-bio.NC: Neurons and Cognition
      - q-bio.OT: Other Quantitative Biology
      - q-bio.PE: Populations and Evolution
      - q-bio.QM: Quantitative Methods
      - q-bio.SC: Subcellular Processes
      - q-bio.TO: Tissues and Organs
      - q-fin.CP: Computational Finance
      - q-fin.EC: Economics
      - q-fin.GN: General Finance
      - q-fin.MF: Mathematical Finance
      - q-fin.PM: Portfolio Management
      - q-fin.PR: Pricing of Securities
      - q-fin.RM: Risk Management
      - q-fin.ST: Statistical Finance
      - q-fin.TR: Trading and Market Microstructure
      - stat.AP: Applications
      - stat.CO: Computation
      - stat.ME: Methodology
      - stat.ML: Machine Learning
      - stat.OT: Other Statistics
      - stat.TH: Statistics Theory
    4.2.2. 根据问题所属领域，将问题拆分成多组关键词的组合，同时组合中的关键词个数尽量不要超过3个
5. Action: 按照下面**回复内容示例**进行回复，注意:
     - 可以向多个查询目标分别查询多次，多个查询用换行分隔，总查询次数控制在5次以内
     - 查询搜索引擎时，需要以"internet:"开头
     - 查询Arxiv论文时，需要以Arxiv的Category值开头，例如"cs.AI:"
     - 查询Arxiv论文时，优先用英文表述关键词进行搜索
     - 当用多个关键词查询时，关键词之间用","分隔
     - 尽量满足**用户发送的消息**中的搜索要求，例如用户要求用英文搜索，则需用英文表述问题和关键词
     - 用户如果没有要求搜索语言，则用和**用户发送的消息**一致的语言表述问题和关键词

### 回复内容示例：

#### 用不同语言查询多次搜索引擎
internet: 黄金价格走势
internet: The trend of gold prices

#### 向Arxiv的多个类目查询多次
cs.AI: attention mechanism
cs.AI: neuron
q-bio.NC: brain,attention mechanism

#### 向多个查询目标查询多次
internet: 中国未来房价趋势
internet: 最新中国经济政策
econ.TH: policy, real estate

# 用户发送的消息为：
{question}
`
	internetSearchPrompts string = `
# 目标
你需要分析**用户发送的消息**，是否需要查询搜索引擎(Google/Bing)，并按照如下情况回复相应内容:

## 情况一：不需要查询搜索引擎
### 情况举例：
1. **用户发送的消息**不是在提问或寻求帮助
2. **用户发送的消息**是要求翻译文字

### 执行指示
根据上面的**情况举例**，如果符合，则按照下面**回复内容**进行回复

### 回复内容：
none

## 情况二：需要查询搜索引擎
### 情况举例：
1. 答复**用户发送的消息**，需依赖互联网上最新的资料
2. 答复**用户发送的消息**，需依赖论文等专业资料
3. 通过查询资料，可以更好地答复**用户发送的消息**

### 执行指示
根据上面的**情况举例**，以及其他需要查询资料的情况，如果符合，按照以下步骤进行处理:
1. What: 分析要答复**用户发送的消息**，需要了解什么知识和资料
2. How: 分析对于要查询的知识和资料，应该提出什么样的问题
3. Adjust: 明确查询什么问题后，用一句话概括问题，并且针对搜索引擎做问题优化
4. Action: 按照下面**回复内容示例**进行回复，注意:
     - 可以查询多次，多个查询用换行分隔，总查询次数控制在5次以内
     - 需要以"internet:"开头
     - 尽量满足**用户发送的消息**中的搜索要求，例如用户要求用英文搜索，则需用英文表述问题和关键词
     - 用户如果没有要求搜索语言，则用和**用户发送的消息**一致的语言表述问题和关键词

### 回复内容示例：

#### 用不同语言查询多次搜索引擎
internet: 黄金价格走势
internet: The trend of gold prices

# 用户发送的消息为：
{question}
`
	privateSearchPrompts string = `
# 目标
你需要分析**用户发送的消息**，是否需要查询搜索引擎(Google/Bing)/私有知识库，并按照如下情况回复相应内容:

## 情况一：不需要查询搜索引擎/私有知识库
### 情况举例：
1. **用户发送的消息**不是在提问或寻求帮助
2. **用户发送的消息**是要求翻译文字

### 执行指示
根据上面的**情况举例**，如果符合，则按照下面**回复内容**进行回复

### 回复内容：
none

## 情况二：需要查询搜索引擎/私有知识库
### 情况举例：
1. 答复**用户发送的消息**，需依赖互联网上最新的资料
2. 答复**用户发送的消息**，需依赖论文等专业资料
3. 通过查询资料，可以更好地答复**用户发送的消息**

### 执行指示
根据上面的**情况举例**，以及其他需要查询资料的情况，如果符合，按照以下步骤进行处理:
1. What: 分析要答复**用户发送的消息**，需要了解什么知识和资料
2. Where: 判断了解这个知识和资料要向Google等搜索引擎提问，还是向私有知识库进行查询，或者需要同时查询多个地方
3. How: 分析对于要查询的知识和资料，应该提出什么样的问题
4. Adjust: 明确要向什么地方查询什么问题后，按下面方式对问题进行调整
  4.1. 向搜索引擎提问：用一句话概括问题，并且针对搜索引擎做问题优化
  4.2. 向私有知识库提问：将问题拆分成多组关键词的组合，同时组合中的关键词个数尽量不要超过3个
5. Action: 按照下面**回复内容示例**进行回复，注意:
     - 可以向多个查询目标分别查询多次，多个查询用换行分隔，总查询次数控制在5次以内
     - 查询搜索引擎时，需要以"internet:"开头
     - 查询私有知识库时，需要以"private:"开头
     - 当用多个关键词查询时，关键词之间用","分隔
     - 尽量满足**用户发送的消息**中的搜索要求，例如用户要求用英文搜索，则需用英文表述问题和关键词
     - 用户如果没有要求搜索语言，则用和**用户发送的消息**一致的语言表述问题和关键词

### 回复内容示例：

#### 用不同语言查询多次搜索引擎
internet: 黄金价格走势
internet: The trend of gold prices

#### 向私有知识库查询多次
private: 电子钱包,密码
private: 张三,身份证号

#### 向多个查询目标查询多次
internet: 中国未来房价趋势
internet: 最新中国经济政策
private: 财务状况

# 用户发送的消息为：
{question}
`
)

func main() {
	wrapper.SetCtx(
		"ai-search",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessStreamingResponseBodyBy(onStreamingResponseBody),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

type searchResult struct {
	title   string
	link    string
	content string
}

func (result searchResult) valid() bool {
	return result.title != "" && result.link != "" && result.content != ""
}

type searchContext struct {
	engineType    string
	querys        []string
	language      string
	arxivCategory string
}

type callArgs struct {
	method             string
	url                string
	headers            [][2]string
	body               []byte
	timeoutMillisecond uint32
}

type searchEngine interface {
	NeedExectue(ctx searchContext) bool
	Client() wrapper.HttpClient
	CallArgs(ctx searchContext) callArgs
	ParseResult(ctx searchContext, response []byte) []searchResult
}

type GoogleSearch struct {
	optionArgs         map[string]string
	apiKey             string
	cx                 string
	start              int
	count              int
	timeoutMillisecond uint32
	client             wrapper.HttpClient
}

func NewGoogleSearch(config *gjson.Result) (*GoogleSearch, error) {
	engine := &GoogleSearch{}
	engine.apiKey = config.Get("apiKey").String()
	if engine.apiKey == "" {
		return nil, errors.New("apiKey not found")
	}
	engine.cx = config.Get("cx").String()
	if engine.cx == "" {
		return nil, errors.New("cx not found")
	}
	serviceName := config.Get("serviceName").String()
	if serviceName == "" {
		return nil, errors.New("serviceName not found")
	}
	servicePort := config.Get("servicePort").Int()
	if servicePort == 0 {
		return nil, errors.New("servicePort not found")
	}
	engine.client = wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: servicePort,
	})
	engine.start = int(config.Get("start").Uint())
	engine.count = int(config.Get("count").Uint())
	if engine.count == 0 {
		engine.count = 10
	}
	if engine.count > 10 || engine.start+engine.count > 100 {
		return nil, errors.New("count must be less than 10, and start + count must be less than or equal to 100.")
	}
	engine.timeoutMillisecond = uint32(config.Get("timeoutMillisecond").Uint())
	if engine.timeoutMillisecond == 0 {
		engine.timeoutMillisecond = 5000
	}
	engine.optionArgs = map[string]string{}
	for key, value := range config.Get("optionArgs").Map() {
		valStr := value.String()
		if valStr != "" {
			engine.optionArgs[key] = value.String()
		}
	}
	return engine, nil
}

func (engine GoogleSearch) NeedExectue(ctx searchContext) bool {
	return ctx.engineType == "internet"
}

func (engine GoogleSearch) Client() wrapper.HttpClient {
	return engine.client
}

func (engine GoogleSearch) CallArgs(ctx searchContext) callArgs {
	queryUrl := fmt.Sprintf("https://customsearch.googleapis.com/customsearch/v1?cx=%s&q=%s&num=%d&key=%s&start=%d",
		engine.cx, url.QueryEscape(strings.Join(ctx.querys, " ")), engine.count, engine.apiKey, engine.start+1)
	var extraArgs []string
	for key, value := range engine.optionArgs {
		extraArgs = append(extraArgs, fmt.Sprintf("%s=%s", key, url.QueryEscape(value)))
	}
	if ctx.language != "" {
		extraArgs = append(extraArgs, fmt.Sprintf("lr=lang_%s", ctx.language))
	}
	if len(extraArgs) > 0 {
		queryUrl = fmt.Sprintf("%s&%s", queryUrl, strings.Join(extraArgs, "&"))
	}
	return callArgs{
		method: http.MethodGet,
		url:    queryUrl,
		headers: [][2]string{
			{"Accept", "application/json"},
		},
		timeoutMillisecond: engine.timeoutMillisecond,
	}
}

func (engine GoogleSearch) ParseResult(ctx searchContext, response []byte) []searchResult {
	jsonObj := gjson.ParseBytes(response)
	var results []searchResult
	for _, item := range jsonObj.Get("items").Array() {
		content := item.Get("snippet").String()
		metaDescription := item.Get("pagemap.metatags.0.og:description").String()
		if metaDescription != "" {
			content = fmt.Sprintf("%s\n...\n%s", content, metaDescription)
		}
		result := searchResult{
			title:   item.Get("title").String(),
			link:    item.Get("link").String(),
			content: content,
		}
		if result.valid() {
			results = append(results, result)
		}
	}
	return results
}

type ArxivSearch struct {
	optionArgs         map[string]string
	start              int
	count              int
	timeoutMillisecond uint32
	client             wrapper.HttpClient
	arxivCategory      string
}

func NewArxivSearch(config *gjson.Result) (*ArxivSearch, error) {
	engine := &ArxivSearch{}
	serviceName := config.Get("serviceName").String()
	if serviceName == "" {
		return nil, errors.New("serviceName not found")
	}
	servicePort := config.Get("servicePort").Int()
	if servicePort == 0 {
		return nil, errors.New("servicePort not found")
	}
	engine.client = wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: servicePort,
	})
	engine.start = int(config.Get("start").Uint())
	engine.count = int(config.Get("count").Uint())
	if engine.count == 0 {
		engine.count = 10
	}
	engine.timeoutMillisecond = uint32(config.Get("timeoutMillisecond").Uint())
	if engine.timeoutMillisecond == 0 {
		engine.timeoutMillisecond = 5000
	}
	engine.optionArgs = map[string]string{}
	for key, value := range config.Get("optionArgs").Map() {
		valStr := value.String()
		if valStr != "" {
			engine.optionArgs[key] = value.String()
		}
	}
	engine.arxivCategory = config.Get("arxivCategory").String()
	return engine, nil
}

func (engine ArxivSearch) NeedExectue(ctx searchContext) bool {
	return ctx.engineType == "arxiv"
}

func (engine ArxivSearch) Client() wrapper.HttpClient {
	return engine.client
}

func (engine ArxivSearch) CallArgs(ctx searchContext) callArgs {
	var searchQueryItems []string
	for _, q := range ctx.querys {
		searchQueryItems = append(searchQueryItems, fmt.Sprintf("all:%s", url.QueryEscape(q)))
	}
	searchQuery := strings.Join(searchQueryItems, "+AND+")
	category := ctx.arxivCategory
	if category == "" {
		category = engine.arxivCategory
	}
	if category != "" {
		searchQuery = fmt.Sprintf("%s+AND+cat:%s", searchQuery, engine.arxivCategory)
	}
	queryUrl := fmt.Sprintf("https://export.arxiv.org/api/query?search_query=%s&max_results=%d&start=%d",
		searchQuery, engine.count, engine.start)
	var extraArgs []string
	for key, value := range engine.optionArgs {
		extraArgs = append(extraArgs, fmt.Sprintf("%s=%s", key, url.QueryEscape(value)))
	}
	if len(extraArgs) > 0 {
		queryUrl = fmt.Sprintf("%s&%s", queryUrl, strings.Join(extraArgs, "&"))
	}
	proxywasm.LogDebugf("ai-search arxiv category:%s, querys:%v, url:%s", category, ctx.querys, queryUrl)
	return callArgs{
		method:             http.MethodGet,
		url:                queryUrl,
		headers:            [][2]string{{"Accept", "application/atom+xml"}},
		timeoutMillisecond: engine.timeoutMillisecond,
	}
}

func (engine ArxivSearch) ParseResult(ctx searchContext, response []byte) []searchResult {
	var results []searchResult
	doc, err := xmlquery.Parse(bytes.NewReader(response))
	if err != nil {
		return results
	}

	entries := xmlquery.Find(doc, "//entry")
	for _, entry := range entries {
		title := entry.SelectElement("title").InnerText()
		link := ""
		for _, l := range entry.SelectElements("link") {
			if l.SelectAttr("rel") == "alternate" && l.SelectAttr("type") == "text/html" {
				link = l.SelectAttr("href")
				break
			}
		}
		summary := entry.SelectElement("summary").InnerText()
		publishTime := entry.SelectElement("published").InnerText()
		authors := entry.SelectElements("author")
		var authorNames []string
		for _, author := range authors {
			authorNames = append(authorNames, author.SelectElement("name").InnerText())
		}
		content := fmt.Sprintf("%s\nAuthors: %s\nPublication time: %s", summary, strings.Join(authorNames, ", "), publishTime)
		result := searchResult{
			title:   title,
			link:    link,
			content: content,
		}
		if result.valid() {
			results = append(results, result)
		}
	}
	return results
}

type BeingSearch struct {
	optionArgs         map[string]string
	apiKey             string
	start              int
	count              int
	timeoutMillisecond uint32
	client             wrapper.HttpClient
}

func NewBeingSearch(config *gjson.Result) (*BeingSearch, error) {
	engine := &BeingSearch{}
	engine.apiKey = config.Get("apiKey").String()
	if engine.apiKey == "" {
		return nil, errors.New("apiKey not found")
	}
	serviceName := config.Get("serviceName").String()
	if serviceName == "" {
		return nil, errors.New("serviceName not found")
	}
	servicePort := config.Get("servicePort").Int()
	if servicePort == 0 {
		return nil, errors.New("servicePort not found")
	}
	engine.client = wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: servicePort,
	})
	engine.start = int(config.Get("start").Uint())
	engine.count = int(config.Get("count").Uint())
	if engine.count == 0 {
		engine.count = 10
	}
	engine.timeoutMillisecond = uint32(config.Get("timeoutMillisecond").Uint())
	if engine.timeoutMillisecond == 0 {
		engine.timeoutMillisecond = 5000
	}
	engine.optionArgs = map[string]string{}
	for key, value := range config.Get("optionArgs").Map() {
		valStr := value.String()
		if valStr != "" {
			engine.optionArgs[key] = value.String()
		}
	}
	return engine, nil
}

func (engine BeingSearch) NeedExectue(ctx searchContext) bool {
	return ctx.engineType == "internet"
}

func (engine BeingSearch) Client() wrapper.HttpClient {
	return engine.client
}

func (engine BeingSearch) CallArgs(ctx searchContext) callArgs {
	queryUrl := fmt.Sprintf("https://api.bing.microsoft.com/v7.0/search?q=%s&count=%d&offset=%d",
		url.QueryEscape(strings.Join(ctx.querys, " ")), engine.count, engine.start)
	var extraArgs []string
	for key, value := range engine.optionArgs {
		extraArgs = append(extraArgs, fmt.Sprintf("%s=%s", key, url.QueryEscape(value)))
	}
	if ctx.language != "" {
		extraArgs = append(extraArgs, fmt.Sprintf("mkt=%s", ctx.language))
	}
	if len(extraArgs) > 0 {
		queryUrl = fmt.Sprintf("%s&%s", queryUrl, strings.Join(extraArgs, "&"))
	}
	return callArgs{
		method:             http.MethodGet,
		url:                queryUrl,
		headers:            [][2]string{{"Ocp-Apim-Subscription-Key", engine.apiKey}},
		timeoutMillisecond: engine.timeoutMillisecond,
	}
}
func (engine BeingSearch) ParseResult(ctx searchContext, response []byte) []searchResult {
	jsonObj := gjson.ParseBytes(response)
	var results []searchResult
	webPages := jsonObj.Get("webPages.value")
	for _, page := range webPages.Array() {
		result := searchResult{
			title:   page.Get("name").String(),
			link:    page.Get("url").String(),
			content: page.Get("snippet").String(),
		}
		if result.valid() {
			results = append(results, result)
		}
		deepLinks := page.Get("deepLinks")
		for _, inner := range deepLinks.Array() {
			innerResult := searchResult{
				title:   inner.Get("name").String(),
				link:    inner.Get("url").String(),
				content: inner.Get("snippet").String(),
			}
			if innerResult.valid() {
				results = append(results, innerResult)
			}
		}
	}
	news := jsonObj.Get("news.value")
	for _, article := range news.Array() {
		result := searchResult{
			title:   article.Get("name").String(),
			link:    article.Get("url").String(),
			content: article.Get("description").String(),
		}
		if result.valid() {
			results = append(results, result)
		}
	}
	return results
}

type SearchRewrite struct {
	client             wrapper.HttpClient
	url                string
	apiKey             string
	modelName          string
	timeoutMillisecond uint32
	prompt             string
}

type Config struct {
	engine          []searchEngine
	promptTemplate  string
	referenceFormat string
	defaultLanguage string
	needReference   bool
	searchRewrite   *SearchRewrite
}

type ElasticsearchSearch struct {
	client             wrapper.HttpClient
	index              string
	contentField       string
	linkField          string
	titleField         string
	start              int
	count              int
	timeoutMillisecond uint32
}

func NewElasticsearchSearch(config *gjson.Result) (*ElasticsearchSearch, error) {
	engine := &ElasticsearchSearch{}
	serviceName := config.Get("serviceName").String()
	if serviceName == "" {
		return nil, errors.New("serviceName not found")
	}
	servicePort := config.Get("servicePort").Int()
	if servicePort == 0 {
		return nil, errors.New("servicePort not found")
	}
	engine.client = wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: servicePort,
	})
	engine.index = config.Get("index").String()
	if engine.index == "" {
		return nil, errors.New("index not found")
	}
	engine.contentField = config.Get("contentField").String()
	if engine.contentField == "" {
		return nil, errors.New("contentField not found")
	}
	engine.linkField = config.Get("linkField").String()
	if engine.linkField == "" {
		return nil, errors.New("linkField not found")
	}
	engine.titleField = config.Get("titleField").String()
	if engine.titleField == "" {
		return nil, errors.New("titleField not found")
	}
	engine.timeoutMillisecond = uint32(config.Get("timeoutMillisecond").Uint())
	if engine.timeoutMillisecond == 0 {
		engine.timeoutMillisecond = 5000
	}
	engine.start = int(config.Get("start").Uint())
	engine.count = int(config.Get("count").Uint())
	if engine.count == 0 {
		engine.count = 10
	}
	return engine, nil
}

func (engine ElasticsearchSearch) NeedExectue(ctx searchContext) bool {
	return ctx.engineType == "private"
}

func (engine ElasticsearchSearch) Client() wrapper.HttpClient {
	return engine.client
}

func (engine ElasticsearchSearch) CallArgs(ctx searchContext) callArgs {
	searchBody := fmt.Sprintf(`{
		"query": {
			"match": {
				"%s": {
					"query": "%s",
					"operator": "AND"
				}
			}
		}
	}`, engine.contentField, strings.Join(ctx.querys, " "))

	return callArgs{
		method: http.MethodPost,
		url:    fmt.Sprintf("/%s/_search?from=%d&size=%d", engine.index, engine.start, engine.count),
		headers: [][2]string{
			{"Content-Type", "application/json"},
		},
		body:               []byte(searchBody),
		timeoutMillisecond: engine.timeoutMillisecond,
	}
}

func (engine ElasticsearchSearch) ParseResult(ctx searchContext, response []byte) []searchResult {
	jsonObj := gjson.ParseBytes(response)
	var results []searchResult
	for _, hit := range jsonObj.Get("hits.hits").Array() {
		source := hit.Get("_source")
		result := searchResult{
			title:   source.Get(engine.titleField).String(),
			link:    source.Get(engine.linkField).String(),
			content: source.Get(engine.contentField).String(),
		}
		if result.valid() {
			results = append(results, result)
		}
	}
	return results
}

func parseConfig(json gjson.Result, config *Config, log wrapper.Log) error {
	config.needReference = json.Get("needReference").Bool()
	if config.needReference {
		config.referenceFormat = json.Get("referenceFormat").String()
		if config.referenceFormat == "" {
			config.referenceFormat = "**References:**\n%s"
		} else if !strings.Contains(config.referenceFormat, "%s") {
			return fmt.Errorf("invalid referenceFormat:%s", config.referenceFormat)
		}
	}
	config.defaultLanguage = json.Get("defaultLang").String()
	config.promptTemplate = json.Get("promptTemplate").String()
	if config.promptTemplate == "" {
		if config.needReference {
			config.promptTemplate = `# 以下内容是基于用户发送的消息的搜索结果:
{search_results}
在我给你的搜索结果中，每个结果都是[webpage X begin]...[webpage X end]格式的，X代表每篇文章的数字索引。请在适当的情况下在句子末尾引用上下文。请按照引用编号[X]的格式在答案中对应部分引用上下文。如果一句话源自多个上下文，请列出所有相关的引用编号，例如[3][5]，切记不要将引用集中在最后返回引用编号，而是在答案对应部分列出。
在回答时，请注意以下几点：
- 今天是北京时间：{cur_date}。
- 并非搜索结果的所有内容都与用户的问题密切相关，你需要结合问题，对搜索结果进行甄别、筛选。
- 对于列举类的问题（如列举所有航班信息），尽量将答案控制在10个要点以内，并告诉用户可以查看搜索来源、获得完整信息。优先提供信息完整、最相关的列举项；如非必要，不要主动告诉用户搜索结果未提供的内容。
- 对于创作类的问题（如写论文），请务必在正文的段落中引用对应的参考编号，例如[3][5]，不能只在文章末尾引用。你需要解读并概括用户的题目要求，选择合适的格式，充分利用搜索结果并抽取重要信息，生成符合用户要求、极具思想深度、富有创造力与专业性的答案。你的创作篇幅需要尽可能延长，对于每一个要点的论述要推测用户的意图，给出尽可能多角度的回答要点，且务必信息量大、论述详尽。
- 如果回答很长，请尽量结构化、分段落总结。如果需要分点作答，尽量控制在5个点以内，并合并相关的内容。
- 对于客观类的问答，如果问题的答案非常简短，可以适当补充一到两句相关信息，以丰富内容。
- 你需要根据用户要求和回答内容选择合适、美观的回答格式，确保可读性强。
- 你的回答应该综合多个相关网页来回答，不能重复引用一个网页。
- 除非用户要求，否则你回答的语言需要和用户提问的语言保持一致。

# 用户消息为：
{question}`
		} else {
			config.promptTemplate = `# 以下内容是基于用户发送的消息的搜索结果:
{search_results}
在我给你的搜索结果中，每个结果都是[webpage begin]...[webpage end]格式的。
在回答时，请注意以下几点：
- 今天是北京时间：{cur_date}。
- 并非搜索结果的所有内容都与用户的问题密切相关，你需要结合问题，对搜索结果进行甄别、筛选。
- 对于列举类的问题（如列举所有航班信息），尽量将答案控制在10个要点以内。如非必要，不要主动告诉用户搜索结果未提供的内容。
- 对于创作类的问题（如写论文），你需要解读并概括用户的题目要求，选择合适的格式，充分利用搜索结果并抽取重要信息，生成符合用户要求、极具思想深度、富有创造力与专业性的答案。你的创作篇幅需要尽可能延长，对于每一个要点的论述要推测用户的意图，给出尽可能多角度的回答要点，且务必信息量大、论述详尽。
- 如果回答很长，请尽量结构化、分段落总结。如果需要分点作答，尽量控制在5个点以内，并合并相关的内容。
- 对于客观类的问答，如果问题的答案非常简短，可以适当补充一到两句相关信息，以丰富内容。
- 你需要根据用户要求和回答内容选择合适、美观的回答格式，确保可读性强。
- 你的回答应该综合多个相关网页来回答，但回答中不要给出网页的引用来源。
- 除非用户要求，否则你回答的语言需要和用户提问的语言保持一致。

# 用户消息为：
{question}`
		}
	}
	if !strings.Contains(config.promptTemplate, "{search_results}") ||
		!strings.Contains(config.promptTemplate, "{question}") {
		return fmt.Errorf("invalid promptTemplate, must contains {search_results} and {question}:%s", config.promptTemplate)
	}
	var internetExists, privateExists, arxivExists bool
	for _, engine := range json.Get("searchFrom").Array() {
		switch engine.Get("type").String() {
		case "being":
			searchEngine, err := NewBeingSearch(&engine)
			if err != nil {
				return fmt.Errorf("being search engine init failed:%s", err)
			}
			config.engine = append(config.engine, searchEngine)
			internetExists = true
		case "google":
			searchEngine, err := NewGoogleSearch(&engine)
			if err != nil {
				return fmt.Errorf("google search engine init failed:%s", err)
			}
			config.engine = append(config.engine, searchEngine)
			internetExists = true
		case "arxiv":
			searchEngine, err := NewArxivSearch(&engine)
			if err != nil {
				return fmt.Errorf("arxiv search engine init failed:%s", err)
			}
			config.engine = append(config.engine, searchEngine)
			arxivExists = true
		case "elasticsearch":
			searchEngine, err := NewElasticsearchSearch(&engine)
			if err != nil {
				return fmt.Errorf("elasticsearch search engine init failed:%s", err)
			}
			config.engine = append(config.engine, searchEngine)
			privateExists = true
		default:
			return fmt.Errorf("unkown search engine:%s", engine.Get("type").String())
		}
	}
	searchRewriteJson := json.Get("searchRewrite")
	if searchRewriteJson.Exists() {
		searchRewrite := &SearchRewrite{}
		llmServiceName := searchRewriteJson.Get("llmServiceName").String()
		if llmServiceName == "" {
			return errors.New("llm_service_name not found")
		}
		llmServicePort := searchRewriteJson.Get("llmServicePort").Int()
		if llmServicePort == 0 {
			return errors.New("llmServicePort not found")
		}
		searchRewrite.client = wrapper.NewClusterClient(wrapper.FQDNCluster{
			FQDN: llmServiceName,
			Port: llmServicePort,
		})
		llmApiKey := searchRewriteJson.Get("llmApiKey").String()
		if llmApiKey == "" {
			return errors.New("llmApiKey not found")
		}
		searchRewrite.apiKey = llmApiKey
		llmUrl := searchRewriteJson.Get("llmUrl").String()
		if llmUrl == "" {
			return errors.New("llmUrl not found")
		}
		searchRewrite.url = llmUrl
		llmModelName := searchRewriteJson.Get("llmModelName").String()
		if llmModelName == "" {
			return errors.New("llmModelName not found")
		}
		searchRewrite.modelName = llmModelName
		llmTimeout := searchRewriteJson.Get("timeoutMillisecond").Uint()
		if llmTimeout == 0 {
			llmTimeout = 10000
		}
		searchRewrite.timeoutMillisecond = uint32(llmTimeout)
		// The consideration here is that internet searches are generally available, but arxiv and private sources may not be.
		if arxivExists {
			if privateExists {
				// private + internet + arxiv
				searchRewrite.prompt = fullSearchPrompts
			} else {
				// internet + arxiv
				searchRewrite.prompt = arxivSearchPrompts
			}
		} else if privateExists {
			// private + internet
			searchRewrite.prompt = privateSearchPrompts
		} else if internetExists {
			// only internet
			searchRewrite.prompt = internetSearchPrompts
		}
		config.searchRewrite = searchRewrite
	}
	if len(config.engine) == 0 {
		return fmt.Errorf("no avaliable search engine found")
	}
	log.Debugf("ai search enabled, config: %#v", config)
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config Config, log wrapper.Log) types.Action {
	contentType, _ := proxywasm.GetHttpRequestHeader("content-type")
	// The request does not have a body.
	if contentType == "" {
		return types.ActionContinue
	}
	if !strings.Contains(contentType, "application/json") {
		log.Warnf("content is not json, can't process: %s", contentType)
		ctx.DontReadRequestBody()
		return types.ActionContinue
	}
	ctx.SetRequestBodyBufferLimit(DEFAULT_MAX_BODY_BYTES)
	_ = proxywasm.RemoveHttpRequestHeader("Accept-Encoding")
	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config Config, body []byte, log wrapper.Log) types.Action {
	var queryIndex int
	var query string
	messages := gjson.GetBytes(body, "messages").Array()
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Get("role").String() == "user" {
			queryIndex = i
			query = messages[i].Get("content").String()
			break
		}
	}
	if query == "" {
		log.Errorf("not found user query in body:%s", body)
		return types.ActionContinue
	}
	searchRewrite := config.searchRewrite
	if searchRewrite != nil {
		startTime := time.Now()
		rewritePrompt := strings.Replace(searchRewrite.prompt, "{question}", query, 1)
		rewriteBody, _ := sjson.SetBytes([]byte(fmt.Sprintf(
			`{"stream":false,"max_tokens":100,"model":"%s","messages":[{"role":"user","content":""}]}`,
			searchRewrite.modelName)), "messages.0.content", rewritePrompt)
		err := searchRewrite.client.Post(searchRewrite.url,
			[][2]string{
				{"Content-Type", "application/json"},
				{"Authorization", fmt.Sprintf("Bearer %s", searchRewrite.apiKey)},
			}, rewriteBody,
			func(statusCode int, responseHeaders http.Header, responseBody []byte) {
				if statusCode != http.StatusOK {
					log.Errorf("search rewrite failed, status: %d", statusCode)
					// After a rewrite failure, no further search is performed, thus quickly identifying the failure.
					proxywasm.ResumeHttpRequest()
					return
				}

				content := gjson.GetBytes(responseBody, "choices.0.message.content").String()
				log.Infof("LLM rewritten query response: %s (took %v), original search query:%s",
					strings.ReplaceAll(content, "\n", `\n`), time.Since(startTime), query)
				if strings.Contains(content, "none") {
					log.Debugf("no search required")
					proxywasm.ResumeHttpRequest()
					return
				}

				// Parse search queries from LLM response
				var searchContexts []searchContext
				for _, line := range strings.Split(content, "\n") {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}

					parts := strings.SplitN(line, ":", 2)
					if len(parts) != 2 {
						continue
					}

					engineType := strings.TrimSpace(parts[0])
					queryStr := strings.TrimSpace(parts[1])

					var ctx searchContext
					ctx.language = config.defaultLanguage

					switch {
					case engineType == "internet":
						ctx.engineType = engineType
						ctx.querys = []string{queryStr}
					case engineType == "private":
						ctx.engineType = engineType
						ctx.querys = strings.Split(queryStr, ",")
						for i := range ctx.querys {
							ctx.querys[i] = strings.TrimSpace(ctx.querys[i])
						}
					default:
						// Arxiv category
						ctx.engineType = "arxiv"
						ctx.arxivCategory = engineType
						ctx.querys = strings.Split(queryStr, ",")
						for i := range ctx.querys {
							ctx.querys[i] = strings.TrimSpace(ctx.querys[i])
						}
					}

					if len(ctx.querys) > 0 {
						searchContexts = append(searchContexts, ctx)
						if ctx.arxivCategory != "" {
							// Conduct inquiries in all areas to increase recall.
							backupCtx := ctx
							backupCtx.arxivCategory = ""
							searchContexts = append(searchContexts, backupCtx)
						}
					}
				}

				if len(searchContexts) == 0 {
					log.Errorf("no valid search contexts found")
					proxywasm.ResumeHttpRequest()
					return
				}
				if types.ActionContinue == executeSearch(ctx, config, queryIndex, body, searchContexts, log) {
					proxywasm.ResumeHttpRequest()
				}
			}, searchRewrite.timeoutMillisecond)
		if err != nil {
			log.Errorf("search rewrite call llm service failed:%s", err)
			// After a rewrite failure, no further search is performed, thus quickly identifying the failure.
			return types.ActionContinue
		}
		return types.ActionPause
	}

	// Execute search without rewrite
	return executeSearch(ctx, config, queryIndex, body, []searchContext{{
		querys:   []string{query},
		language: config.defaultLanguage,
	}}, log)
}

func executeSearch(ctx wrapper.HttpContext, config Config, queryIndex int, body []byte, searchContexts []searchContext, log wrapper.Log) types.Action {
	var searchResultGroups [][]searchResult = make([][]searchResult, len(config.engine))
	var finished int
	var searching int
	for i := 0; i < len(config.engine); i++ {
		engine := config.engine[i]

		// Check if engine needs to execute for any of the search contexts
		var needsExecute bool
		for _, searchCtx := range searchContexts {
			if engine.NeedExectue(searchCtx) {
				needsExecute = true
				break
			}
		}
		if !needsExecute {
			continue
		}

		// Process all search contexts for this engine
		for _, searchCtx := range searchContexts {
			if !engine.NeedExectue(searchCtx) {
				continue
			}
			args := engine.CallArgs(searchCtx)
			index := i
			err := engine.Client().Call(args.method, args.url, args.headers, args.body,
				func(statusCode int, responseHeaders http.Header, responseBody []byte) {
					defer func() {
						finished++
						if finished == searching {
							// Merge search results from all engines
							var mergedResults []searchResult
							for _, results := range searchResultGroups {
								mergedResults = append(mergedResults, results...)
							}
							// Format search results for prompt template
							var formattedResults []string
							var formattedReferences []string
							for j, result := range mergedResults {
								if config.needReference {
									formattedResults = append(formattedResults,
										fmt.Sprintf("[webpage %d begin]\n%s\n[webpage %d end]", j+1, result.content, j+1))
									formattedReferences = append(formattedReferences,
										fmt.Sprintf("[%d] [%s](%s)", j+1, result.title, result.link))
								} else {
									formattedResults = append(formattedResults,
										fmt.Sprintf("[webpage begin]\n%s\n[webpage end]", result.content))
								}
							}
							// Prepare template variables
							curDate := time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006年1月2日")
							searchResults := strings.Join(formattedResults, "\n")
							log.Debugf("searchResults: %s", searchResults)
							// Fill prompt template
							prompt := strings.Replace(config.promptTemplate, "{search_results}", searchResults, 1)
							prompt = strings.Replace(prompt, "{question}", searchContexts[0].querys[0], 1)
							prompt = strings.Replace(prompt, "{cur_date}", curDate, 1)
							// Update request body with processed prompt
							modifiedBody, err := sjson.SetBytes(body, fmt.Sprintf("messages.%d.content", queryIndex), prompt)
							if err != nil {
								log.Errorf("modify request message content failed, err:%v, body:%s", err, body)
							} else {
								log.Debugf("modifeid body:%s", modifiedBody)
								proxywasm.ReplaceHttpRequestBody(modifiedBody)
								if config.needReference {
									ctx.SetContext("References", strings.Join(formattedReferences, "\n"))
								}
							}
							proxywasm.ResumeHttpRequest()
						}
					}()
					if statusCode != http.StatusOK {
						log.Errorf("search call failed, status: %d, engine: %#v", statusCode, engine)
						return
					}
					// Append results to existing slice for this engine
					searchResultGroups[index] = append(searchResultGroups[index], engine.ParseResult(searchCtx, responseBody)...)
				}, args.timeoutMillisecond)
			if err != nil {
				log.Errorf("search call failed, engine: %#v", engine)
				continue
			}
			searching++
		}
	}
	if searching > 0 {
		return types.ActionPause
	}
	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config Config, log wrapper.Log) types.Action {
	if !config.needReference {
		ctx.DontReadResponseBody()
		return types.ActionContinue
	}
	proxywasm.RemoveHttpResponseHeader("content-length")
	contentType, err := proxywasm.GetHttpResponseHeader("Content-Type")
	if err != nil || !strings.HasPrefix(contentType, "text/event-stream") {
		if err != nil {
			log.Errorf("unable to load content-type header from response: %v", err)
		}
		ctx.BufferResponseBody()
		ctx.SetResponseBodyBufferLimit(DEFAULT_MAX_BODY_BYTES)
	}
	return types.ActionContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, config Config, body []byte, log wrapper.Log) types.Action {
	references := ctx.GetStringContext("References", "")
	if references == "" {
		return types.ActionContinue
	}
	content := gjson.GetBytes(body, "choices.0.message.content")
	modifiedContent := fmt.Sprintf("%s\n\n%s", fmt.Sprintf(config.referenceFormat, references), content)
	body, err := sjson.SetBytes(body, "choices.0.message.content", modifiedContent)
	if err != nil {
		log.Errorf("modify response message content failed, err:%v, body:%s", err, body)
		return types.ActionContinue
	}
	proxywasm.ReplaceHttpResponseBody(body)
	return types.ActionContinue
}

func onStreamingResponseBody(ctx wrapper.HttpContext, config Config, chunk []byte, isLastChunk bool, log wrapper.Log) []byte {
	if ctx.GetBoolContext("ReferenceAppended", false) {
		return chunk
	}
	references := ctx.GetStringContext("References", "")
	if references == "" {
		return chunk
	}
	modifiedChunk, responseReady := setReferencesToFirstMessage(ctx, chunk, fmt.Sprintf(config.referenceFormat, references), log)
	if responseReady {
		ctx.SetContext("ReferenceAppended", true)
		return modifiedChunk
	} else {
		return []byte("")
	}
}

const PARTIAL_MESSAGE_CONTEXT_KEY = "partialMessage"

func setReferencesToFirstMessage(ctx wrapper.HttpContext, chunk []byte, references string, log wrapper.Log) ([]byte, bool) {
	if len(chunk) == 0 {
		log.Debugf("chunk is empty")
		return nil, false
	}

	var partialMessage []byte
	partialMessageI := ctx.GetContext(PARTIAL_MESSAGE_CONTEXT_KEY)
	if partialMessageI != nil {
		if pMsg, ok := partialMessageI.([]byte); ok {
			partialMessage = append(pMsg, chunk...)
		} else {
			log.Warnf("invalid partial message type: %T", partialMessageI)
			partialMessage = chunk
		}
	} else {
		partialMessage = chunk
	}

	if len(partialMessage) == 0 {
		log.Debugf("partial message is empty")
		return nil, false
	}
	messages := strings.Split(string(partialMessage), "\n\n")
	if len(messages) > 1 {
		firstMessage := messages[0]
		log.Debugf("first message: %s", firstMessage)
		firstMessage = strings.TrimPrefix(firstMessage, "data: ")
		firstMessage = strings.TrimSuffix(firstMessage, "\n")
		deltaContent := gjson.Get(firstMessage, "choices.0.delta.content")
		modifiedMessage, err := sjson.Set(firstMessage, "choices.0.delta.content", fmt.Sprintf("%s\n\n%s", references, deltaContent))
		if err != nil {
			log.Errorf("modify response delta content failed, err:%v", err)
			return partialMessage, true
		}
		modifiedMessage = fmt.Sprintf("data: %s", modifiedMessage)
		log.Debugf("modified message: %s", firstMessage)
		messages[0] = string(modifiedMessage)
		return []byte(strings.Join(messages, "\n\n")), true
	}
	ctx.SetContext(PARTIAL_MESSAGE_CONTEXT_KEY, partialMessage)
	return nil, false
}
