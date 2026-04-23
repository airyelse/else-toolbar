export namespace envvars {
	
	export class EnvVar {
	    name: string;
	    value: string;
	    isPath: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EnvVar(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.value = source["value"];
	        this.isPath = source["isPath"];
	    }
	}
	export class EnvResult {
	    system: EnvVar[];
	    user: EnvVar[];
	
	    static createFrom(source: any = {}) {
	        return new EnvResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.system = this.convertValues(source["system"], EnvVar);
	        this.user = this.convertValues(source["user"], EnvVar);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class MCPSkillResult {
	    mcps: opencode.MCPInfo[];
	    skills: opencode.SkillInfo[];
	
	    static createFrom(source: any = {}) {
	        return new MCPSkillResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mcps = this.convertValues(source["mcps"], opencode.MCPInfo);
	        this.skills = this.convertValues(source["skills"], opencode.SkillInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace models {
	
	export class CategoryDTO {
	    id: number;
	    name: string;
	    parentId?: number;
	    icon: string;
	    order: number;
	    children: CategoryDTO[];
	
	    static createFrom(source: any = {}) {
	        return new CategoryDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.parentId = source["parentId"];
	        this.icon = source["icon"];
	        this.order = source["order"];
	        this.children = this.convertValues(source["children"], CategoryDTO);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TagDTO {
	    id: number;
	    name: string;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new TagDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.color = source["color"];
	    }
	}
	export class EntryDTO {
	    id: number;
	    title: string;
	    username: string;
	    password?: string;
	    url: string;
	    categoryId?: number;
	    categoryName: string;
	    tagIds: number[];
	    tags: TagDTO[];
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new EntryDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.url = source["url"];
	        this.categoryId = source["categoryId"];
	        this.categoryName = source["categoryName"];
	        this.tagIds = source["tagIds"];
	        this.tags = this.convertValues(source["tags"], TagDTO);
	        this.notes = source["notes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace opencode {
	
	export class AgentConfig {
	    model: any;
	    variant?: string;
	    skills?: string[];
	    mcps?: string[];
	    temperature?: number;
	
	    static createFrom(source: any = {}) {
	        return new AgentConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.variant = source["variant"];
	        this.skills = source["skills"];
	        this.mcps = source["mcps"];
	        this.temperature = source["temperature"];
	    }
	}
	export class AppendPromptDiff {
	    agent: string;
	    store: string;
	    file: string;
	
	    static createFrom(source: any = {}) {
	        return new AppendPromptDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agent = source["agent"];
	        this.store = source["store"];
	        this.file = source["file"];
	    }
	}
	export class MCPInfo {
	    name: string;
	    type: string;
	    command: string;
	    url: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new MCPInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.command = source["command"];
	        this.url = source["url"];
	        this.source = source["source"];
	    }
	}
	export class Preset {
	    orchestrator?: AgentConfig;
	    oracle?: AgentConfig;
	    librarian?: AgentConfig;
	    explorer?: AgentConfig;
	    designer?: AgentConfig;
	    fixer?: AgentConfig;
	
	    static createFrom(source: any = {}) {
	        return new Preset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.orchestrator = this.convertValues(source["orchestrator"], AgentConfig);
	        this.oracle = this.convertValues(source["oracle"], AgentConfig);
	        this.librarian = this.convertValues(source["librarian"], AgentConfig);
	        this.explorer = this.convertValues(source["explorer"], AgentConfig);
	        this.designer = this.convertValues(source["designer"], AgentConfig);
	        this.fixer = this.convertValues(source["fixer"], AgentConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PresetDiff {
	    store_active: string;
	    file_active: string;
	    differences: string[];
	
	    static createFrom(source: any = {}) {
	        return new PresetDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.store_active = source["store_active"];
	        this.file_active = source["file_active"];
	        this.differences = source["differences"];
	    }
	}
	export class PresetStoreData {
	    active_preset: string;
	    presets: Record<string, Preset>;
	
	    static createFrom(source: any = {}) {
	        return new PresetStoreData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active_preset = source["active_preset"];
	        this.presets = this.convertValues(source["presets"], Preset, true);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SkillInfo {
	    name: string;
	    description: string;
	    source: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.source = source["source"];
	    }
	}

}

export namespace pathenv {
	
	export class PathEntry {
	    rawPath: string;
	    path: string;
	    exists: boolean;
	    isDir: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PathEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rawPath = source["rawPath"];
	        this.path = source["path"];
	        this.exists = source["exists"];
	        this.isDir = source["isDir"];
	    }
	}
	export class PathProfileDTO {
	    name: string;
	    paths: string[];
	
	    static createFrom(source: any = {}) {
	        return new PathProfileDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.paths = source["paths"];
	    }
	}
	export class PathResult {
	    system: PathEntry[];
	    user: PathEntry[];
	
	    static createFrom(source: any = {}) {
	        return new PathResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.system = this.convertValues(source["system"], PathEntry);
	        this.user = this.convertValues(source["user"], PathEntry);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace runtime {
	
	export class Config {
	    baseDir: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.baseDir = source["baseDir"];
	    }
	}
	export class SDKVersion {
	    version: string;
	    path: string;
	    active: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SDKVersion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.path = source["path"];
	        this.active = source["active"];
	    }
	}
	export class SDKInfo {
	    type: string;
	    name: string;
	    icon: string;
	    installed: SDKVersion[];
	    current: string;
	
	    static createFrom(source: any = {}) {
	        return new SDKInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.name = source["name"];
	        this.icon = source["icon"];
	        this.installed = this.convertValues(source["installed"], SDKVersion);
	        this.current = source["current"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

