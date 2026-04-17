import { Injectable } from "@nestjs/common";
import { AuthDto } from "./dto/auth.dto";

@Injectable()
export class AuthService {
    sayHello(): string {
        return "Hello World";
    }

    signUp(bodyMessage: AuthDto) {
        return {
            email:  bodyMessage.email,
            password: bodyMessage.password, 
        }
    }
}