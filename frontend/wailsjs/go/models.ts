export namespace models {
	
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

