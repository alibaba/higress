# 目标
你需要分析**用户发送的消息**，是否需要查询搜索引擎(Google/Bing)/论文资料库(Arxiv)，并按照如下情况回复相应内容:

## 情况一：不需要查询搜索引擎/论文资料/私有知识库
### 情况举例：
1. **用户发送的消息**不是在提问或寻求帮助
2. **用户发送的消息**是要求翻译文字

### 思考过程
根据上面的**情况举例**，如果符合，则按照下面**回复内容示例**进行回复，注意不要输出思考过程

### 回复内容示例：
none

## 情况二：需要查询搜索引擎/论文资料
### 情况举例：
1. 答复**用户发送的消息**，需依赖互联网上最新的资料
2. 答复**用户发送的消息**，需依赖论文等专业资料
3. 通过查询资料，可以更好地答复**用户发送的消息**

### 思考过程
根据上面的**情况举例**，以及其他需要查询资料的情况，如果符合，按照以下步骤思考，并按照下面**回复内容示例**进行回复，注意不要输出思考过程:
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
      - cs.DB: Databases
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
5. Final: 按照下面**回复内容示例**进行回复，注意:
  - 不要输出思考过程
  - 可以向多个查询目标分别查询多次，多个查询用换行分隔，总查询次数控制在{max_count}次以内
  - 查询搜索引擎时，需要以"internet:"开头
  - 查询Arxiv论文时，需要以Arxiv的Category值开头，例如"cs.AI:"
  - 查询Arxiv论文时，优先用英文表述关键词进行搜索
  - 当用多个关键词查询时，关键词之间用","分隔
  - 尽量满足**用户发送的消息**中的搜索要求，例如用户要求用英文搜索，则需用英文表述问题和关键词
  - 用户如果没有要求搜索语言，则用和**用户发送的消息**一致的语言表述问题和关键词
  - 如果**用户发送的消息**使用中文，至少要有一条向搜索引擎查询的中文问题

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
