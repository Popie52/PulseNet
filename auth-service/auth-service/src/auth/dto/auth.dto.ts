import { IsString, IsNotEmpty, IsEmail, IsStrongPassword } from "class-validator";


export class SignupDto {    
    @IsEmail()
    @IsNotEmpty()
    email: string

    @IsStrongPassword({
        minLength: 8,
        minLowercase: 1,
        minUppercase: 1,
        minSymbols: 1,
        minNumbers: 1,
    })
    password: string 
}

