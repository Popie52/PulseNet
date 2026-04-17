import { Injectable } from "@nestjs/common";
import { SignupDto } from "./dto/auth.dto";
import * as bcrypt from "bcrypt";
import { JwtService } from "@nestjs/jwt";
import { access } from "fs";

@Injectable()
export class AuthService {

    constructor(private jwtService: JwtService) {}

    sayHello(): string {
        return "Hello World";
    }

    async signUp(data: SignupDto) {
        const hashedPassword = await bcrypt.hash(data.password, 10);
        return {
            email:  data.email,
            password: hashedPassword, 
        }
    }

    async login(data: SignupDto) {
        const payload = data;

        const storedUser = {
            email: "test123@gmail.com",
            password: await bcrypt.hash("12345678aA@#", 10),
        };

        const isMatch = await bcrypt.compare(payload.password, storedUser.password);
        
        if(!isMatch) {
            throw new Error("Invalid credentials");
        }

        const token = this.jwtService.sign(payload.email);

        return {
            access_token: token,
        }
    }
}