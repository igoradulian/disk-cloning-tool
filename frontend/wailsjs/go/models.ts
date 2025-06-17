export namespace models {
	
	export class CopyProgress {
	    sourceDisk: string;
	    targetDisks: string[];
	    bytesCopied: number;
	    totalBytes: number;
	    progress: number;
	    speed: number;
	    timeRemaining: number;
	    status: string;
	    // Go type: time
	    startTime: any;
	    md5Hash: string;
	    sha256Hash: string;
	    sha1Hash: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new CopyProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceDisk = source["sourceDisk"];
	        this.targetDisks = source["targetDisks"];
	        this.bytesCopied = source["bytesCopied"];
	        this.totalBytes = source["totalBytes"];
	        this.progress = source["progress"];
	        this.speed = source["speed"];
	        this.timeRemaining = source["timeRemaining"];
	        this.status = source["status"];
	        this.startTime = this.convertValues(source["startTime"], null);
	        this.md5Hash = source["md5Hash"];
	        this.sha256Hash = source["sha256Hash"];
	        this.sha1Hash = source["sha1Hash"];
	        this.error = source["error"];
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
	export class DiskInfo {
	    path: string;
	    name: string;
	    size: number;
	    sectorSize: number;
	    serialNumber: string;
	    model: string;
	    isRemovable: boolean;
	    isReadOnly: boolean;
	    fileSystem: string;
	
	    static createFrom(source: any = {}) {
	        return new DiskInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.sectorSize = source["sectorSize"];
	        this.serialNumber = source["serialNumber"];
	        this.model = source["model"];
	        this.isRemovable = source["isRemovable"];
	        this.isReadOnly = source["isReadOnly"];
	        this.fileSystem = source["fileSystem"];
	    }
	}

}

