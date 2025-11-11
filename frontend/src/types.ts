export type RepositoryResponse = {
    name: string,
    members: Member[],
    files: S3File[],
    userPermission: "owner" | "none" |"full" | "read"
}
export type Repository = {
    name: string,
    userPermission: "owner" | "none" |"full" | "read"
}
type Member = {
    id: number,
    username: string,
    permission: "" | "full" | "read"
}
export type S3File = {
    id: number,
    ownerUsername: string,
    path: string,
    type: "file" | "folder",
    size: number,
    uploadDate: number
}
